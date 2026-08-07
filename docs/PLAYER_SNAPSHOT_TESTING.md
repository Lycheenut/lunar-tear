# 线上玩家存档导出到本地测试

线上 VM 只允许通过云控制台内置的 SSH Shell 访问。因此先在 VM 中生成快照文件，再使用控制台的文件下载功能取回本地；不需要从本地电脑连接 VM。

导出结果会自动改成本地用户 `1`，并清除账号 UUID、外部账号绑定、好友、生日、充值额、昵称和玩家自定义文本。关卡、任务、背包、角色、武器和宝石等游戏状态会保留。

只需注意两件事：

- 不要把快照提交到 Git；本地的 `server/snapshots/` 已被 `.gitignore` 忽略。
- 导入会直接覆盖本地 `db/game.db` 中的玩家 `1`；想保留原存档时自行先备份。

## 1. 在云控制台导出并下载

生产镜像需要已经包含 `./export-snapshot`。本仓库的 `server/Dockerfile` 会自动构建它。

打开云平台控制台提供的 SSH Shell，确认游戏内的 `player_id` 和服务器上的项目目录，然后在 VM 中执行：

```sh
cd /path/to/lunar-tear/server
PLAYER_ID=12345
EXPORT_DIR="$HOME/lunar-tear-exports"
EXPORT_FILE="$EXPORT_DIR/player-${PLAYER_ID}.json"

umask 077
mkdir -p "$EXPORT_DIR"

if LUNAR_ADMIN_TOKEN=unused docker compose \
  --env-file deploy/.env.production \
  -f deploy/docker-compose.production.yaml \
  exec -T server ./export-snapshot \
  --db db/game.db \
  --player-id "$PLAYER_ID" > "$EXPORT_FILE" \
  && test -s "$EXPORT_FILE"; then
  sha256sum "$EXPORT_FILE"
  realpath "$EXPORT_FILE"
else
  rm -f -- "$EXPORT_FILE"
  echo "export failed"
fi
```

`LUNAR_ADMIN_TOKEN=unused` 只用于让 Docker Compose 解析配置，不会替换运行中容器的密钥。导出程序把 JSON 写入 VM 文件，把摘要写到终端，不会停止或修改线上服务。

导出成功后，终端最后会打印文件的绝对路径，例如：

```text
/home/example-user/lunar-tear-exports/player-12345.json
```

在控制台 SSH Shell 的菜单中选择 **Download file / 下载文件**，输入这条绝对路径，将 JSON 下载到本地。确认本地文件可用后，回到控制台删除 VM 上的临时文件：

```sh
rm -f -- "$HOME/lunar-tear-exports/player-12345.json"
```

玩家刚好在操作时，极低概率会得到跨表时间点略有差异的快照。通常直接使用即可；如果出现明显不合理的数据，让玩家退出游戏后再导出一次。只有重复出现问题时才考虑短暂停服。

## 2. 快速检查文件

macOS：

```bash
cd /path/to/lunar-tear/server
SNAPSHOT="$HOME/Downloads/player-12345.json"

python3 -m json.tool "$SNAPSHOT" >/dev/null
shasum -a 256 "$SNAPSHOT"
```

Windows PowerShell：

```powershell
Set-Location C:\path\to\lunar-tear\server
$Snapshot = "$HOME\Downloads\player-12345.json"

Get-Content -Raw -LiteralPath $Snapshot | ConvertFrom-Json | Out-Null
(Get-FileHash -Algorithm SHA256 -LiteralPath $Snapshot).Hash.ToLowerInvariant()
```

可选抽查：`UserId` 和 `PlayerId` 应为 `1`，`Profile.Name` 应为 `Local Test Player`。

## 3. 覆盖本地玩家 1

本地代码和 master data 最好与线上版本一致。导入工具会覆盖默认 `db/game.db` 中的玩家 `1`，同时保留该玩家当前的客户端 UUID。

Windows PowerShell：

```powershell
make stop
make import SNAPSHOT="$Snapshot"
make start
```

macOS：

```bash
make stop
make import SNAPSHOT="$SNAPSHOT"
make start
```

客户端重新连接后应进入 `Local Test Player`。确认关键等级、任务进度和背包内容后即可开始复现。

如果本地还没有玩家 `1`，先正常启动一次服务并让客户端注册，然后再执行以上覆盖步骤。

## 4. 用完清理

测试结束后删除下载的快照即可。本地玩家 `1` 会保留测试后的状态；如需恢复，使用导入前自行保存的本地数据库备份。

## 常见问题

- `find player ... store: not found`：传入的应是游戏内 `player_id`，不是昵称。
- `--uuid is required because local user 1 does not exist`：先正常启动本地服务，让客户端注册玩家 `1`。
- 导入后内容明显不对：先检查本地代码和 master data 是否与线上版本一致；如果玩家导出时正在操作，让玩家退出后重导一次。
