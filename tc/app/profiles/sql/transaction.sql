{{define "_queryGlobalTransactionDO"}}
    select
    xid,
    transaction_id,
    status,
    application_id,
    transaction_service_group,
    transaction_name,
    timeout,
    begin_time,
    application_data,
    gmt_create,
    gmt_modified
    from global_table
{{end}}

{{define "_queryGlobalTransactionDOByXid"}}
    select
    xid,
    transaction_id,
    status,
    application_id,
    transaction_service_group,
    transaction_name,
    timeout,
    begin_time,
    application_data,
    gmt_create,
    gmt_modified
    from global_table
    where xid = ?
{{end}}

{{define "_queryGlobalTransactionDOByTransactionId"}}
    select
    xid,
    transaction_id,
    status,
    application_id,
    transaction_service_group,
    transaction_name,
    timeout,
    begin_time,
    application_data,
    gmt_create,
    gmt_modified
    from global_table
    where transaction_id = ?
{{end}}

{{define "_insertGlobalTransactionDO"}}
    insert into global_table
    (xid,
    transaction_id,
    status,
    application_id,
    transaction_service_group,
    transaction_name,
    timeout,
    begin_time,
    application_data,
    gmt_create,
    gmt_modified)
    values(?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
{{end}}

{{define "_updateGlobalTransactionDO"}}
    update global_table
    set
        status = ?,
        gmt_modified = now()
    where xid = ?
{{end}}

{{define "_deleteGlobalTransactionDO"}}
    delete from global_table
    where xid = ?
{{end}}

{{define "_queryBranchTransactionDO"}}
    select
    `xid`,
    `branch_id`,
    `transaction_id`,
    `resource_group_id`,
    `resource_id`,
    `branch_type`,
    `status`,
    `client_id`,
    `application_data`,
    `gmt_create`,
    `gmt_modified`
    from branch_table
{{end}}

{{define "_queryBranchTransactionDOByXid"}}
    select
    xid,
    branch_id,
    transaction_id,
    resource_group_id,
    resource_id,
    branch_type,
    status,
    client_id,
    application_data,
    gmt_create,
    gmt_modified
    from branch_table
    where xid = ?
    order by gmt_create asc
{{end}}

{{define "_insertBranchTransactionDO"}}
    insert into branch_table
    (xid,
    branch_id,
    transaction_id,
    resource_group_id,
    resource_id,
    branch_type,
    status,
    client_id,
    application_data,
    gmt_create,
    gmt_modified)
    values(?, ?, ?, ?, ?, ?, ?, ?, ?, now(6), now(6))
{{end}}

{{define "_updateBranchTransactionDO"}}
    update branch_table
    set
        status = ?,
        gmt_modified = now(6)
    where xid = ? and branch_id = ?
{{end}}

{{define "_deleteBranchTransactionDO"}}
    delete from branch_table
    where xid = ? and branch_id = ?
{{end}}

{{define "_queryMaxTransactionId"}}
    select max(transaction_id) as maxTransactioId from global_table
    where transaction_id < ? and transaction_id > ?
{{end}}

{{define "_queryMaxBranchId"}}
    select max(branch_id) as maxBranchId from branch_table
    where branch_id < ? and branch_id > ?
{{end}}