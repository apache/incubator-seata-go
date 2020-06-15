package cache

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/xiaobudongzhang/seata-golang/base/sql_type"

	_struct "github.com/xiaobudongzhang/seata-golang/client/at/sql/struct"
	"github.com/xiaobudongzhang/seata-golang/pkg/logging"
)

var EXPIRE_TIME = 900 * 1000 * time.Microsecond

type Tx struct {
	*sql.Tx
	config *mysql.Config
}

func newTx(tx *sql.Tx, dsn string) *Tx {
	config, _ := mysql.ParseDSN(dsn)
	return &Tx{
		Tx:     tx,
		config: config,
	}
}

type MysqlTableMetaCache struct {
	tableMetaCache *cache.Cache
	dsn            string
}

func NewMysqlTableMetaCache(dsn string) ITableMetaCache {
	return &MysqlTableMetaCache{
		tableMetaCache: cache.New(EXPIRE_TIME, 10*EXPIRE_TIME),
		dsn:            dsn,
	}
}

func (cache *MysqlTableMetaCache) GetTableMeta(tx *sql.Tx, tableName, resourceId string) (_struct.TableMeta, error) {
	if tableName == "" {
		return _struct.TableMeta{}, errors.New("TableMeta cannot be fetched without tableName")
	}
	cacheKey := cache.GetCacheKey(tableName, resourceId)
	tMeta, found := cache.tableMetaCache.Get(cacheKey)
	if found {
		meta := tMeta.(_struct.TableMeta)
		return meta, nil
	} else {
		ndb := newTx(tx, cache.dsn)
		meta, err := cache.FetchSchema(ndb, tableName)
		if err != nil {
			return _struct.TableMeta{}, errors.WithStack(err)
		}
		cache.tableMetaCache.Set(cacheKey, meta, EXPIRE_TIME)
		return meta, nil
	}
}

func (cache *MysqlTableMetaCache) Refresh(tx *sql.Tx, resourceId string) {
	for k, v := range cache.tableMetaCache.Items() {
		meta := v.Object.(_struct.TableMeta)
		key := cache.GetCacheKey(meta.TableName, resourceId)
		if k == key {
			ndb := newTx(tx, cache.dsn)
			tMeta, err := cache.FetchSchema(ndb, meta.TableName)
			if err != nil {
				logging.Logger.Errorf("get table meta error:%s", err.Error())
			}
			if !cmp.Equal(tMeta, meta) {
				cache.tableMetaCache.Set(key, tMeta, EXPIRE_TIME)
				logging.Logger.Info("table meta change was found, update table meta cache automatically.")
			}
		}
	}
}

func (cache *MysqlTableMetaCache) GetCacheKey(tableName string, resourceId string) string {
	var defaultTableName string
	tableNameWithCatalog := strings.Split(strings.ReplaceAll(tableName, "`", ""), ".")
	if len(tableNameWithCatalog) > 1 {
		defaultTableName = tableNameWithCatalog[1]
	} else {
		defaultTableName = tableNameWithCatalog[0]
	}
	return fmt.Sprintf("%s.%s", resourceId, defaultTableName)
}

func (cache *MysqlTableMetaCache) FetchSchema(tx *Tx, tableName string) (_struct.TableMeta, error) {
	tm := _struct.TableMeta{TableName: tableName,
		AllColumns: make(map[string]_struct.ColumnMeta),
		AllIndexes: make(map[string]_struct.IndexMeta),
	}
	columns, err := GetColumns(tx, tableName)
	if err != nil {
		return _struct.TableMeta{}, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
	}
	for _, column := range columns {
		tm.AllColumns[column.ColumnName] = column
	}
	indexes, err := GetIndexes(tx, tableName)
	if err != nil {
		return _struct.TableMeta{}, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
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
		return _struct.TableMeta{}, errors.Errorf("Could not found any index in the table: %s", tableName)
	}

	return tm, nil
}

func GetColumns(tx *Tx, tableName string) ([]_struct.ColumnMeta, error) {
	var tn = tableName
	if strings.Contains(tableName, ".") {
		idx := strings.LastIndex(tableName, ".")
		tName := tableName[idx+1:]
		tn = strings.Trim(tName, "`")
	}
	args := []interface{}{tx.config.DBName, tn}
	//`TABLE_CATALOG`,	`TABLE_SCHEMA`,	`TABLE_NAME`,	`COLUMN_NAME`,	`ORDINAL_POSITION`,	`COLUMN_DEFAULT`,
	//`IS_NULLABLE`, `DATA_TYPE`,	`CHARACTER_MAXIMUM_LENGTH`,	`CHARACTER_OCTET_LENGTH`,	`NUMERIC_PRECISION`,
	//`NUMERIC_SCALE`, `DATETIME_PRECISION`, `CHARACTER_SET_NAME`,	`COLLATION_NAME`,	`COLUMN_TYPE`,	`COLUMN_KEY',
	//`EXTRA`,	`PRIVILEGES`, `COLUMN_COMMENT`, `GENERATION_EXPRESSION`, `SRS_ID`
	s := "SELECT `TABLE_CATALOG`, `TABLE_SCHEMA`, `TABLE_NAME`, `COLUMN_NAME`, `DATA_TYPE`, `CHARACTER_MAXIMUM_LENGTH`, " +
		"`NUMERIC_PRECISION`, `NUMERIC_SCALE`, `IS_NULLABLE`, `COLUMN_COMMENT`, `COLUMN_DEFAULT`, `CHARACTER_OCTET_LENGTH`, " +
		"`ORDINAL_POSITION`, `COLUMN_KEY`, `EXTRA`  FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	rows, err := tx.Query(s, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]_struct.ColumnMeta, 0)
	for rows.Next() {
		col := _struct.ColumnMeta{}

		var tableCat, tableSchem, tableName, columnName, dataType, isNullable, remark, colDefault, colKey, extra sql.NullString
		var columnSize, decimalDigits, numPreRadix, charOctetLength, ordinalPosition sql.NullInt32
		err = rows.Scan(&tableCat, &tableSchem, &tableName, &columnName,
			&dataType, &columnSize, &decimalDigits, &numPreRadix, &isNullable,
			&remark, &colDefault, &charOctetLength, &ordinalPosition, &colKey, &extra)
		if err != nil {
			return nil, err
		}
		col.TableCat = tableCat.String
		col.TableSchemeName = tableSchem.String
		col.TableName = tableName.String
		col.ColumnName = strings.Trim(columnName.String, "` ")
		col.DataTypeName = dataType.String
		col.DataType = sql_type.GetSqlType(dataType.String)
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
		col.IsAutoIncrement = extra.String

		result = append(result, col)
	}
	return result, nil
}

func GetIndexes(tx *Tx, tableName string) ([]_struct.IndexMeta, error) {
	var tn = tableName
	if strings.Contains(tableName, ".") {
		idx := strings.LastIndex(tableName, ".")
		tName := tableName[idx+1:]
		tn = strings.Trim(tName, "`")
	}
	args := []interface{}{tx.config.DBName, tn}

	//`TABLE_CATALOG`, `TABLE_SCHEMA`, `TABLE_NAME`, `NON_UNIQUE`, `INDEX_SCHEMA`, `INDEX_NAME`, `SEQ_IN_INDEX`,
	//`COLUMN_NAME`, `COLLATION`, `CARDINALITY`, `SUB_PART`, `PACKED`, `NULLABLE`, `INDEX_TYPE`, `COMMENT`,
	//`INDEX_COMMENT`, `IS_VISIBLE`, `EXPRESSION`
	s := "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `INDEX_TYPE`, `SEQ_IN_INDEX`, `COLLATION`, `CARDINALITY` " +
		"FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	rows, err := tx.Query(s, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]_struct.IndexMeta, 0)
	for rows.Next() {
		index := _struct.IndexMeta{
			Values: make([]_struct.ColumnMeta, 0),
		}
		var indexName, columnName, nonUnique, indexType, collation sql.NullString
		var ordinalPosition, cardinality sql.NullInt32
		err = rows.Scan(&indexName, &columnName, &nonUnique, &indexType, &ordinalPosition, &collation, &cardinality)

		index.IndexName = indexName.String
		index.ColumnName = columnName.String
		if "yes" == strings.ToLower(nonUnique.String) || nonUnique.String == "1" {
			index.NonUnique = true
		}
		index.OrdinalPosition = ordinalPosition.Int32
		index.AscOrDesc = collation.String
		index.Cardinality = cardinality.Int32
		if "primary" == strings.ToLower(indexName.String) {
			index.IndexType = _struct.IndexType_PRIMARY
		} else if !index.NonUnique {
			index.IndexType = _struct.IndexType_UNIQUE
		} else {
			index.IndexType = _struct.IndexType_NORMAL
		}

		result = append(result, index)
	}
	return result, nil
}
