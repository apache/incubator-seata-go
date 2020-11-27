# 示例运行说明
1. 执行 `/scripts/server/db/mysql.sql`
2. 执行 `./scrpts/seata_order.sql`
3. 执行 `./scrpts/seata_product.sql`
4. 编译 `/tc/app/cmd、./order_svc、./product_svc、./aggregation_svc`
5. 修改 `/tc/app/cmd/profiles/dev/config.yml` 中的dsn配置
6. 修改 `./order_svc、./product_svc`两服务的 `conf/client.yml` 中的dsn配置
7. 运行 `./cmd start -config ../profiles/dev/config.yml`
8. 分别运行 `./order_svc、./product_svc、./aggregation_svc`
9. 测试 commit: `http://localhost:8003/createSoCommit`
10. 测试 rollback: `http://localhost:8003/createSoRollback`
11. That's All, Happy Coding!!!

PS:
```
   / 代表到从系统根目录到seata-golang目录的绝对路径
   . 代表此README.md所在的目录
```
    