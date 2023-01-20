## Prepare the environment
- go test environment: go1.18 windows/amd64
- The following instructions are performed on the win64 command line

```
cd /d C:\Users\Administrator\Desktop\
git clone https://github.com/seata/seata-go.git
cd /d C:\Users\Administrator\Desktop\seata-go
```

## Run example
### 1. basic
```
go run sample\at_zorm\basic\main.go ^
    sample\at_zorm\basic\service.go 
```

### 2. gin
server
```
go run sample\at_zorm\gin\server\main.go ^
    sample\at_zorm\gin\server\service.go
```
client
```
go run sample\at_zorm\gin\client\main.go
```

### 3. non_transaction
```
go run sample\at_zorm\non_transaction\main.go ^
    sample\at_zorm\non_transaction\service.go
```

### 4. rollback (distributed transaction)
server
```
go run sample\at_zorm\rollback\server\main.go ^
    sample\at_zorm\rollback\server\service.go ^
    sample\at_zorm\rollback\server\zormGlobalTransactionPlugin.go
```

server2
```
go run sample\at_zorm\rollback\server2\main.go ^
    sample\at_zorm\rollback\server2\service.go ^
    sample\at_zorm\rollback\server2\zormGlobalTransactionPlugin.go
```

client
```
go run sample\at_zorm\rollback\client\main.go 
```

## Documentation on zorm
- [zorm.cn](https://zorm.cn/)
- [springrain \/ zorm](https://gitee.com/chunanyong/zorm)
- [zorm-examples](https://gitee.com/wuxiangege/zorm-examples/)