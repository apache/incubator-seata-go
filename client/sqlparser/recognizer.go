package sqlparser

type SQLType byte

const (
	SQLType_SELECT SQLType = iota

	SQLType_INSERT

	SQLType_UPDATE

	SQLType_DELETE

	SQLType_SELECT_FOR_UPDATE

	SQLType_REPLACE

	SQLType_TRUNCATE

	SQLType_CREATE

	SQLType_DROP

	SQLType_LOAD

	SQLType_MERGE

	SQLType_SHOW

	SQLType_ALTER

	SQLType_RENAME

	SQLType_DUMP

	SQLType_DEBUG

	SQLType_EXPLAIN

	SQLType_PROCEDURE

	SQLType_DESC

    // ******************************************
	// 一些 java mybatis 特有的 sql 方法省略
	// ******************************************

	SQLType_SET SQLType = 27

	SQLType_RELOAD

	SQLType_SELECT_UNION

	SQLType_CREATE_TABLE

	SQLType_DROP_TABLE

	SQLType_ALTER_TABLE

	SQLType_SAVE_POINT

	SQLType_SELECT_FROM_UPDATE

	SQLType_MULTI_DELETE

	SQLType_MULTI_UPDATE

	SQLType_CREATE_INDEX

	SQLType_DROP_INDEX

	// ******************************************
	// 一些不常见的 sql 类型省略，确有需要，以后再加
	// ******************************************
)

type IParametersHolder interface {
	GetParameters() []interface{}
}

type ISQLRecognizer interface {
	/**
	 * Type of the SQL. INSERT/UPDATE/DELETE ...
	 *
	 * @return sql type
	 */
	GetSQLType() SQLType

	/**
	 * TableRecords source related in the SQL, including alias if any.
	 * SELECT id, name FROM user u WHERE ...
	 * Alias should be 'u' for this SQL.
	 *
	 * @return table source.
	 */
	GetTableAlias() string

	/**
	 * TableRecords name related in the SQL.
	 * SELECT id, name FROM user u WHERE ...
	 * TableRecords name should be 'user' for this SQL, without alias 'u'.
	 *
	 * @return table name.
	 * @see #getTableAlias() #getTableAlias()#getTableAlias()
	 */
	GetTableName() string

	/**
	 * Return the original SQL input by the upper application.
	 *
	 * @return The original SQL.
	 */
	GetOriginalSQL() string
}

type IWhereRecognizer interface {
	ISQLRecognizer

	/**
	 * Gets where condition.
	 *
	 * @return the where condition
	 */
	GetWhereCondition() string
}

type ISQLSelectRecognizer interface {
	IWhereRecognizer
}

type ISQLInsertRecognizer interface {
	ISQLRecognizer

	/**
	 * Gets insert columns.
	 *
	 * @return the insert columns
	 */
	GetInsertColumns() []string

	/**
	 * Gets insert rows.
	 *
	 * @return the insert rows
	 */
	GetInsertRows() [][]string
}

type ISQLDeleteRecognizer interface {
	IWhereRecognizer
}

type ISQLUpdateRecognizer interface {
	IWhereRecognizer

	/**
	 * Gets update columns.
	 *
	 * @return the update columns
	 */
	GetUpdateColumns() []string

	/**
	 * Gets update values.
	 *
	 * @return the update values
	 */
	GetUpdateValues() []string
}