## 用例介绍
此用例介绍如何在tcc本地模式下使用防悬挂功能

## 使用步骤

- 在您的数据库中使用``./sample/tcc/fence/script/mysql.sql``脚本创建防悬挂所需的日志记录表，如果您使用的是其他数据库则运行对应数据库的脚本文件。
- 在``./sample/tcc/fence/service/service.go``中修改数据库驱动名为对应数据库类型并引入相关驱动包，mysql无需修改。此外需要注意用户名和密码是否正确。
- 启动``seata tc server``
- 使用以下命令运行用例``go run ./sample/tcc/fence/cmd/main.go``