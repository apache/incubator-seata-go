package postgresql

import (
	"database/sql"
	"fmt"
	"strings"
)
import (
	"github.com/google/go-cmp/cmp"
	"github.com/lib/pq"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

//可以根据导入的meta_cache在对应导入其他扩展点
import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema"
	tableMetaCache "github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache"
	_ "github.com/transaction-wg/seata-golang/pkg/client/at/undo/manager/postgres"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	sql2 "github.com/transaction-wg/seata-golang/pkg/util/sql"
	stringUtil "github.com/transaction-wg/seata-golang/pkg/util/string"
)

type PostgresqlTableMetaCache struct {
	TableMetaCache *cache.Cache
	Dsn            string
}

func NewPostgresqlTableMetaCache(dsn string) tableMetaCache.ITableMetaCache {
	tableMetaCache := cache.New(tableMetaCache.EXPIRE_TIME, 10*tableMetaCache.EXPIRE_TIME)
	return &PostgresqlTableMetaCache{
		TableMetaCache: tableMetaCache,
		Dsn:            dsn,
	}
}

func (cache *PostgresqlTableMetaCache) GetTableMeta(tx *sql.Tx, tableName, resourceID string) (schema.TableMeta, error) {
	if tableName == "" {
		return schema.TableMeta{}, errors.New("TableMeta cannot be fetched without tableName")
	}
	cacheKey := cache.GetCacheKey(tableName, resourceID)
	tMeta, found := cache.TableMetaCache.Get(cacheKey)
	if found {
		meta := tMeta.(schema.TableMeta)
		return meta, nil
	} else {
		ndb := newTx(tx, cache.Dsn)
		meta, err := cache.FetchSchema(ndb, tableName)
		if err != nil {
			return schema.TableMeta{}, errors.WithStack(err)
		}
		cache.TableMetaCache.Set(cacheKey, meta, tableMetaCache.EXPIRE_TIME)
		return meta, nil
	}
}
func (cache *PostgresqlTableMetaCache) Refresh(tx *sql.Tx, resourceID string) {
	for k, v := range cache.TableMetaCache.Items() {
		meta := v.Object.(schema.TableMeta)
		key := cache.GetCacheKey(meta.TableName, resourceID)
		if k == key {
			ndb := newTx(tx, cache.Dsn)
			tMeta, err := cache.FetchSchema(ndb, meta.TableName)
			if err != nil {
				log.Errorf("get table meta error:%s", err.Error())
			}
			if !cmp.Equal(tMeta, meta) {
				cache.TableMetaCache.Set(key, tMeta, tableMetaCache.EXPIRE_TIME)
				log.Info("table meta change was found, update table meta cache automatically.")
			}
		}
	}
}

func (cache *PostgresqlTableMetaCache) GetCacheKey(tableName string, resourceID string) string {
	var defaultTableName string
	tableNameWithCatalog := strings.Split(strings.ReplaceAll(tableName, "`", ""), ".")
	if len(tableNameWithCatalog) > 1 {
		defaultTableName = tableNameWithCatalog[1]
	} else {
		defaultTableName = tableNameWithCatalog[0]
	}
	return fmt.Sprintf("%s.%s", resourceID, defaultTableName)
}
func (cache *PostgresqlTableMetaCache) FetchSchema(tx *Tx, tableName string) (schema.TableMeta, error) {
	tm := schema.TableMeta{TableName: tableName,
		AllColumns: make(map[string]schema.ColumnMeta),
		AllIndexes: make(map[string]schema.IndexMeta),
	}
	columnMetas, err := GetColumns(tx, tableName)
	if err != nil {
		return schema.TableMeta{}, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
	}
	columns := make([]string, 0)
	for _, column := range columnMetas {
		tm.AllColumns[column.ColumnName] = column
		columns = append(columns, column.ColumnName)
	}
	tm.Columns = columns
	indexes, err := GetIndexes(tx, tableName)
	if err != nil {
		return schema.TableMeta{}, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
	}
	for _, index := range indexes {
		col := tm.AllColumns[index.ColumnName]
		idx, ok := tm.AllIndexes[index.IndexName]
		if ok {
			idx.Values = append(idx.Values, col)
		} else {
			index.Values = append(index.Values, col)
			tm.AllIndexes[index.IndexName] = index
		}
	}
	if len(tm.AllIndexes) == 0 {
		return schema.TableMeta{}, errors.Errorf("Could not found any index in the table: %s", tableName)
	}

	return tm, nil
}

func GetColumns(tx *Tx, tableName string) ([]schema.ColumnMeta, error) {
	var tn = stringUtil.Escape(tableName, "`")
	args := []interface{}{tx.GetSchema(), tn}
	//POSTGRESQL查找表列信息sql语法
	s := "SELECT TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, " +
		"NUMERIC_PRECISION, NUMERIC_SCALE, IS_NULLABLE, COLUMN_DEFAULT, CHARACTER_OCTET_LENGTH, " +
		"ORDINAL_POSITION FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = $1 AND TABLE_NAME = $2"

	rows, err := tx.Query(s, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]schema.ColumnMeta, 0)
	for rows.Next() {
		col := schema.ColumnMeta{}

		var tableCat, tableSchem, tableName, columnName, dataType, isNullable, remark, colDefault sql.NullString
		var columnSize, decimalDigits, numPreRadix, charOctetLength, ordinalPosition sql.NullInt32
		err = rows.Scan(&tableCat, &tableSchem, &tableName, &columnName,
			&dataType, &columnSize, &decimalDigits, &numPreRadix, &isNullable,
			&colDefault, &charOctetLength, &ordinalPosition)
		if err != nil {
			return nil, err
		}
		col.TableCat = tableCat.String
		col.TableSchemeName = tableSchem.String
		col.TableName = tableName.String
		col.ColumnName = strings.Trim(columnName.String, "` ")
		col.DataTypeName = dataType.String
		col.DataType = sql2.GetSqlType(dataType.String)
		col.ColumnSize = columnSize.Int32
		col.DecimalDigits = decimalDigits.Int32
		col.NumPrecRadix = numPreRadix.Int32
		col.IsNullable = isNullable.String
		if strings.ToLower(isNullable.String) == "yes" {
			col.Nullable = 1
		} else {
			col.Nullable = 0
		}
		col.Remarks = remark.String
		col.ColumnDef = colDefault.String
		col.SqlDataType = 0
		col.SqlDatetimeSub = 0
		col.CharOctetLength = charOctetLength.Int32
		col.OrdinalPosition = ordinalPosition.Int32

		result = append(result, col)
	}
	return result, nil
}

func GetIndexes(tx *Tx, tableName string) ([]schema.IndexMeta, error) {
	var tn = stringUtil.Escape(tableName, "`")
	args := []interface{}{tx.GetSchema(), tn}
	//pgsql查找索引信息
	//todo 更好的查找索引方式？
	s := `select
    i.relname as index_name,
    a.attname as column_name,
    pi.indexdef as index_info
from
    pg_class t,
    pg_class i,
    pg_index ix,
		pg_indexes pi,
    pg_attribute a
where
    t.oid = ix.indrelid
    and i.oid = ix.indexrelid
    and a.attrelid = t.oid
    and a.attnum = ANY(ix.indkey)
    and t.relkind = 'r'
		and pi.indexname = i.relname
		and pi.schemaname = $1
		and t.relname = $2
order by
    t.relname,
    i.relname`

	rows, err := tx.Query(s, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]schema.IndexMeta, 0)
	for rows.Next() {
		index := schema.IndexMeta{
			Values: make([]schema.ColumnMeta, 0),
		}
		var indexName, columnName, indexInfo sql.NullString
		err = rows.Scan(&indexName, &columnName, &indexInfo)

		index.IndexName = indexName.String
		index.ColumnName = columnName.String
		if !strings.Contains(indexInfo.String, "UNIQUE") {
			index.NonUnique = true
		}
		if strings.Contains(indexInfo.String, "pkey") {
			index.IndexType = schema.IndexType_PRIMARY
		} else if !index.NonUnique {
			index.IndexType = schema.IndexType_UNIQUE
		} else {
			index.IndexType = schema.IndexType_NORMAL
		}
		result = append(result, index)
	}
	return result, nil
}

type Tx struct {
	*sql.Tx
	config string
}

func (tx *Tx) GetSchema() string {
	schema := ConvertKVStringToMap(tx.config)["search_path"]
	return stringUtil.Escape(schema, "'")
}
func ConvertKVStringToMap(str string) map[string]string {
	data := strings.Fields(str)
	res := make(map[string]string)
	for _, kvStr := range data {
		kv := strings.Split(kvStr, "=")
		k := kv[0]
		v := kv[1]
		res[k] = v
	}
	return res
}
func newTx(tx *sql.Tx, dsn string) *Tx {
	config, _ := pq.ParseURL(dsn)
	return &Tx{
		Tx:     tx,
		config: config,
	}
}
