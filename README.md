# FileInPic

一个简单的文件分享服务。

## 使用方法
### 安装

```sh
curl -o deploy.sh https://raw.githubusercontent.com/zouzonghao/fileinpic/refs/heads/main/deploy.sh && chmod +x deploy.sh && ./deploy.sh install
```

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
# 用于API请求的API密钥
api_key: "PASSWORD"
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
export API_KEY="PASSWORD"
./fileinpic
```

如果同时提供了配置文件和环境变量，则配置文件中的值将覆盖环境变量。
## API 使用

### 认证

要使用 API，您需要通过 `X-API-KEY` 请求头提供 API 密钥。您可以在 `config.yaml` 文件中或通过 `API_KEY` 环境变量设置 API 密钥。

### 上传文件

要上传文件，请向 `/api/v1/files/upload` 端点发送 `POST` 请求，请求体中包含文件的二进制数据。

**必需的请求头:**

*   `X-API-KEY`: 您的 API 密钥。
*   `Content-Disposition`: `attachment; filename="your_file_name"`

**使用 curl 的示例:**

```bash
curl -X POST \
  -H "X-API-KEY: PASSWORD" \
  -H "Content-Disposition: attachment; filename=\"test.txt\"" \
  --data-binary "@path/to/your/file" \
  http://localhost:37374/api/v1/files/upload
```

**成功响应:**

```json
{
  "ok": true,
  "url": "/api/v1/files/public/download/1"
}
```

### 下载文件

要下载文件，您可以使用上传响应中返回的公共 URL。下载文件不需要认证。

**使用 curl 的示例:**

```bash
curl -X GET \
  -o "downloaded_file" \
  http://localhost:37374/api/v1/files/public/download/1
```

### 删除文件

要删除文件，请向 `/api/v1/files/delete/{id}` 端点发送 `DELETE` 请求，其中 `{id}` 是您要删除的文件的 ID。

**必需的请求头:**

*   `X-API-KEY`: 您的 API 密钥。

**使用 curl 的示例:**

```bash
curl -X DELETE \
  -H "X-API-KEY: PASSWORD" \
  http://localhost:37374/api/v1/files/delete/1
```