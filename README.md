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
## API Usage

### Authentication

To use the API, you need to provide an API key via the `X-API-KEY` header. You can set the API key in the `config.yaml` file or via the `API_KEY` environment variable.

### Upload a file

To upload a file, send a `POST` request to the `/api/v1/files/upload` endpoint with the file's binary data in the request body.

**Required headers:**

*   `X-API-KEY`: Your API key.
*   `Content-Disposition`: `attachment; filename="your_file_name"`

**Example using curl:**

```bash
curl -X POST \
  -H "X-API-KEY: PASSWORD" \
  -H "Content-Disposition: attachment; filename=\"test.txt\"" \
  --data-binary "@path/to/your/file" \
  http://localhost:37374/api/v1/files/upload
```

**Success response:**

```json
{
  "ok": true,
  "url": "/api/v1/files/public/download/1"
}
```

### Download a file

To download a file, you can use the public URL returned in the upload response. No authentication is required to download the file.

**Example using curl:**

```bash
curl -X GET \
  -o "downloaded_file" \
  http://localhost:37374/api/v1/files/public/download/1
```

### Delete a file

To delete a file, send a `DELETE` request to the `/api/v1/files/delete/{id}` endpoint, where `{id}` is the ID of the file you want to delete.

**Required headers:**

*   `X-API-KEY`: Your API key.

**Example using curl:**

```bash
curl -X DELETE \
  -H "X-API-KEY: PASSWORD" \
  http://localhost:37374/api/v1/files/delete/1