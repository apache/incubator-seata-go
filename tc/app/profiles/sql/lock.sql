{{define "_queryLockDO"}}
    select
    row_key,
    xid,
    transaction_id,
    branch_id,
    resource_id,
    table_name,
    pk,
    gmt_create,
    gmt_modified
    from lock_table
{{end}}

{{define "_batchDeleteLock"}}
    delete from lock_table
    where xid = ? AND INCONDITIONSTR
{{end}}

{{define "_batchDeleteLockByBranch"}}
    delete from lock_table
    where xid = ? AND branch_id = ?
{{end}}

{{define "_getLockDOCount"}}
    select
    count(1) as total
    from lock_table
{{end}}