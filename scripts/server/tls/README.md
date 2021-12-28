# 如何在seata-golang中使用TLS

#### 1，修改 openssl.cnf 文件中的 **[alt_names]** 标签，新增或修改 **DNS** 的内容，作为客户端访问的 **ServerName** 

#### 2，运行 **create_keys** 脚本，过程中可以填写必要的信息或全部回车跳过

#### 3，运行成功后会生成： ca.crt、 ca.csr、 ca.key、 ca.srl、 server.csr、server.key、server.pem

#### 4，开启TC的TLS：

在TC的配置文件中修改配置：

```yaml
serverTLS:
  enable: true
  certFilePath: {server.pem文件路径}
  keyFilePath: {server.key文件路径}
```



#### 5，开启TM和RM的TLS：

在TM和RM的配置文件中修改配置：

```yaml
clientTLS:
  enable: true
  certFilePath: {server.pem文件路径}
  serverName: {openssl.cnf文件中[alt_names]标签的DNS内容，例如："test.seata.io"}
```



