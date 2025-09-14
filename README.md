# FileInPic

一个简单的文件分享服务。

## 使用方法

### 构建

```bash
go build -o fileinpic .
```

### 运行

您可以使用YAML文件或环境变量来配置应用程序。

#### 使用配置文件

创建一个 `config.yaml` 文件：

```yaml
# 服务器地址, 例如 "http://localhost:37374"
host: ""
# 登录密码
password: "admin"
# 用于API请求的认证令牌
auth_token: ""
```

然后运行应用程序：

```bash
./fileinpic -config config.yaml
```

#### 使用环境变量

```bash
export PASSWORD="your_password"
export HOST="http://localhost:37374"
export AUTH_TOKEN="your_secret_token"
./fileinpic
```

如果同时提供了配置文件和环境变量，则配置文件中的值将覆盖环境变量。