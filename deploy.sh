#!/bin/bash

# --- 配置 ---
APP_NAME="fileinpic"
INSTALL_DIR="/opt/$APP_NAME"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"
TMP_DIR=$(mktemp -d -t fileinpic-install-XXXXXXXX) # 创建一个安全的临时目录

# --- 脚本健壮性设置 ---

# 1. 如果任何命令失败，立即退出
set -e

# 2. 设置一个标志，用于判断安装过程是否完成
_INSTALL_COMPLETE=false

# 3. 定义清理函数，在脚本退出时自动调用
cleanup() {
  # 仅当安装过程未完成时，才执行回滚
  if [ "$_INSTALL_COMPLETE" = "false" ]; then
    echo "错误：安装过程未能完成。正在回滚并清理..." >&2
    
    # 尝试停止并禁用服务
    if systemctl is-active --quiet "$APP_NAME.service" &>/dev/null; then
      systemctl stop "$APP_NAME.service"
    fi
    if systemctl is-enabled --quiet "$APP_NAME.service" &>/dev/null; then
      systemctl disable "$APP_NAME.service"
    fi

    # 删除安装过程中创建的文件
    if [ -f "$SERVICE_FILE" ]; then
      rm -f "$SERVICE_FILE"
      systemctl daemon-reload
    fi
    if [ -d "$INSTALL_DIR" ]; then
      rm -rf "$INSTALL_DIR"
    fi
  fi
  
  # 无论成功与否，总是删除临时目录
  if [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

# 4. 注册 trap，确保无论脚本如何退出（正常、错误、中断），都会调用 cleanup 函数
trap cleanup EXIT

# --- 主函数 ---

# 检查脚本是否以 root 权限运行
check_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "错误：此脚本必须以 root 权限运行。" >&2
    exit 1
  fi
}

# 安装函数
do_install() {
    echo "开始安装 $APP_NAME..."

    # 检查是否已安装
    if [ -d "$INSTALL_DIR" ] || [ -f "$SERVICE_FILE" ]; then
        echo "警告: 检测到 $APP_NAME 的现有安装。"
        read -p "您想继续并覆盖现有安装吗？ (y/N): " choice
        case "$choice" in
          y|Y )
            # 询问是否重置配置文件
            read -p "是否重置配置文件 (config.yaml)？ (y/N): " reset_config
            # 询问是否删除数据库
            read -p "是否删除数据库文件 (fileinpic.db)？ (y/N): " delete_db
            
            # 确定是否保留文件
            local keep_config="keep"
            local keep_db="keep"
            
            if [ "$reset_config" = "y" ] || [ "$reset_config" = "Y" ]; then
                keep_config=""
            fi
            
            if [ "$delete_db" = "y" ] || [ "$delete_db" = "Y" ]; then
                keep_db=""
            fi
            
            echo "正在继续安装，将首先卸载旧版本..."
            do_uninstall "silent" "$keep_config" "$keep_db"
            ;;
          * )
            echo "安装已取消。"
            exit 0
            ;;
        esac
    fi

    # --- 文件安装阶段开始 ---
    
    # 1. 获取下载链接
    read -p "请输入 fileinpic-linux-amd64.tar.gz 的下载链接: " DOWNLOAD_URL
    if [ -z "$DOWNLOAD_URL" ]; then
        echo "错误：下载链接不能为空。" >&2
        exit 1
    fi

    # 2. 下载并解压
    echo "正在下载文件到 $TMP_DIR..."
    wget -O "$TMP_DIR/fileinpic-linux-amd64.tar.gz" "$DOWNLOAD_URL"

    echo "正在解压文件..."
    tar -xzf "$TMP_DIR/fileinpic-linux-amd64.tar.gz" -C "$TMP_DIR"
    if [ ! -f "$TMP_DIR/$APP_NAME" ] || [ ! -d "$TMP_DIR/static" ]; then
        echo "错误：压缩包内容不正确。应包含 '$APP_NAME' 可执行文件和 'static' 目录。" >&2
        exit 1
    fi

    # 3. 创建安装目录并移动文件
    echo "正在创建安装目录并移动文件..."
    mkdir -p "$INSTALL_DIR"
    mv "$TMP_DIR/$APP_NAME" "$INSTALL_DIR/"
    mv "$TMP_DIR/static" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$APP_NAME"

    # 4. 获取用户配置并生成 config.yaml
    if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
        echo "请输入配置信息以生成 config.yaml:"
        read -p "Host (例如: 0.0.0.0:8080): " HOST
        read -sp "Password: " PASSWORD
        echo
        read -p "Auth Token: " AUTH_TOKEN

        echo "正在生成 config.yaml..."
        cat > "$INSTALL_DIR/config.yaml" << EOF
host: "$HOST"
password: "$PASSWORD"
auth_token: "$AUTH_TOKEN"
EOF
    else
        echo "检测到现有配置文件，将跳过重新配置。"
    fi

    # 5. 创建 systemd 服务
    echo "正在创建 systemd 服务..."
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=FileInPic Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$APP_NAME -config $INSTALL_DIR/config.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    # 6. 重载并启用服务
    echo "正在重载 systemd 并启用服务..."
    systemctl daemon-reload
    systemctl enable "$APP_NAME.service"

    # --- 文件安装阶段结束 ---
    _INSTALL_COMPLETE=true # 标记安装过程已完成，从现在起即使失败也不再回滚
    echo "文件安装成功。"

    # --- 服务启动阶段 ---
    echo "正在尝试启动服务..."
    # 尝试启动服务，但不使用 set -e，这样即使失败脚本也不会立即退出
    if ! systemctl start "$APP_NAME.service"; then
        echo "----------------------------------------" >&2
        echo "错误：服务启动失败！" >&2
        echo "安装文件已保留在 $INSTALL_DIR 中供您调试。" >&2
        echo "请使用以下命令检查详细的错误日志：" >&2
        echo "journalctl -u $APP_NAME.service -l --no-pager" >&2
        echo "----------------------------------------" >&2
        exit 1
    fi

    # 延迟一小段时间，然后检查服务状态
    sleep 2
    if ! systemctl is-active --quiet "$APP_NAME.service"; then
        echo "----------------------------------------" >&2
        echo "错误：服务启动后未能保持运行状态！" >&2
        echo "安装文件已保留在 $INSTALL_DIR 中供您调试。" >&2
        echo "请使用以下命令检查详细的错误日志：" >&2
        echo "journalctl -u $APP_NAME.service -l --no-pager" >&2
        echo "----------------------------------------" >&2
        exit 1
    fi

    echo "----------------------------------------"
    echo "$APP_NAME 安装并启动成功！"
    echo "服务状态:"
    systemctl status "$APP_NAME.service" --no-pager
    echo "----------------------------------------"
}

# 卸载函数
do_uninstall() {
    local mode=$1 # "silent" or empty
    local keep_config=$2 # "keep" to keep config file
    local keep_db=$3 # "keep" to keep database file
    
    # 保存要保留的文件
    local config_backup=""
    local db_backup=""
    
    if [ "$keep_config" = "keep" ] && [ -f "$INSTALL_DIR/config.yaml" ]; then
        echo "备份配置文件..."
        cp "$INSTALL_DIR/config.yaml" "$TMP_DIR/config.yaml.backup"
        config_backup="$TMP_DIR/config.yaml.backup"
    fi
    
    if [ "$keep_db" = "keep" ] && [ -f "$INSTALL_DIR/fileinpic.db" ]; then
        echo "备份数据库文件..."
        cp "$INSTALL_DIR/fileinpic.db" "$TMP_DIR/fileinpic.db.backup"
        db_backup="$TMP_DIR/fileinpic.db.backup"
    fi
    
    if [ "$mode" != "silent" ]; then
        echo "开始卸载 $APP_NAME..."
    fi

    # 1. 停止并禁用服务
    echo "正在停止并禁用 systemd 服务..."
    systemctl stop "$APP_NAME.service" || true
    systemctl disable "$APP_NAME.service" || true

    # 2. 删除服务文件
    if [ -f "$SERVICE_FILE" ]; then
        echo "正在删除 systemd 服务文件..."
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
    fi

    # 3. 删除安装目录
    if [ -d "$INSTALL_DIR" ]; then
        echo "正在删除安装目录..."
        rm -rf "$INSTALL_DIR"
    fi
    
    # 恢复保留的文件
    if [ -n "$config_backup" ] || [ -n "$db_backup" ]; then
        echo "恢复保留的文件..."
        mkdir -p "$INSTALL_DIR"
        
        if [ -n "$config_backup" ]; then
            echo "恢复配置文件..."
            cp "$config_backup" "$INSTALL_DIR/config.yaml"
        fi
        
        if [ -n "$db_backup" ]; then
            echo "恢复数据库文件..."
            cp "$db_backup" "$INSTALL_DIR/fileinpic.db"
        fi
    fi

    if [ "$mode" != "silent" ]; then
        echo "$APP_NAME 卸载完成。"
    fi
}

# --- 脚本入口 ---
main() {
    check_root
    case "$1" in
        install)
            do_install
            ;;
        uninstall)
            do_uninstall "" "" ""
            ;;
        *)
            echo "用法: $0 {install|uninstall}"
            exit 1
            ;;
    esac
    exit 0
}

main "$@"