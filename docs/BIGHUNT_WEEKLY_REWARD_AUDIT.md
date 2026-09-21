# 讨伐战每周奖励检查（2026-09-21）

结论：现有逻辑不完整。已在本地修复待领取状态缺失、奖励档位累加和结算分数返回错误。尚未部署，也未修改线上玩家数据。

## 玩家 2 的线上证据

- 用户确认玩家 2 在 9 月 14–20 日（PST）实际打过讨伐战。
- Cloud Logging 显示玩家 2 在 `2026-09-21 08:49:56 UTC`（PST 00:49:56）执行 `GameStart`，随后正常请求登录奖励。
- 查询 `08:00–12:16 UTC` 的游戏服日志，没有发现 `ReceiveBigHuntReward` 或 `GetBigHuntTopData` 调用。不是该时段内已经出现的领奖 RPC 报错。
- 未读取线上玩家数据库，故不能断言玩家 2 的具体周成绩、领取标记或应发数量。代码缺陷与目前的线上症状一致；最终个案确认仍需核对这些记录和实际部署版本。

## 已确认并修复的问题

### 1. 缺少待领取记录，客户端不会发起领奖

`FinishBigHuntQuest` 只保存 `BigHuntWeeklyMaxScores`。`BigHuntWeeklyStatuses` 原先只在领奖成功后写入 `true`，从未随周成绩创建待领取的 `false` 记录。

原客户端 `CalculatorBigHuntQuest.HasRewardReceiveBigHunt`（ARM64 RVA `0x2C6CAC4`）读取 `IUserBigHuntWeeklyStatus`，列表为空或相应记录已领取时直接返回 `false`。它并不会仅凭周成绩发现奖励。因此直接调用领奖服务的原有单测能够通过，真实客户端却不会发送请求。

修复：正式战斗写入成绩后创建缺失的周状态；登录时根据已有的正分周成绩补齐缺失记录。已有状态及领取时间保持原值。新测试通过真实 SQLite、`Auth → GetUserData → ReceiveBigHuntReward` 验证发现与领取，并验证再次登录、再次领取不会重复到账。

### 2. 周奖励错误地累计所有达标档位

周奖励此前调用 `CollectNewRewards(group, 0, score)`，把所有正分达标档位相加，并漏掉门槛为 0 的基础档。

原客户端 `GetWeeklyAttributeRewardPossessionItem`（RVA `0x2C6E4D4`）对 `NecessaryScore <= score` 的档位按门槛降序排列，再取 `FirstOrDefault`，即每个属性只取最高达标档位。其过滤和排序回调分别为 `0x29F73BC`、`0x2C6FA84`。

本地主数据也证实各档是完整奖励：属性 3、组 2042 的 2,000 分档有物品 30 × 71，4,000 分档有物品 30 × 72。旧实现给 4,000 分发 143，正确值是 72。这是本地主数据的示例，不代表已核算玩家 2 的线上奖励。

修复：按属性结算一次，只选择最高达标档，支持 0 分门槛；无该周成绩的属性不发奖。测试覆盖基础档、多档、超过最高档、重复属性和重复领取。

### 3. 结算分数全部误用本周分数

`GetBigHuntTopData` 和 `ReceiveBigHuntReward` 原先把 `BeforeMaxScore`、`AfterMaxScore`、`CurrentMaxScore` 全部填成本周分数，周一尚未挑战时会全部返回 0。

原客户端 `GetBigHuntAttributeData`（RVA `0x2C6DB48`）分别从前周、上周的周成绩和当前赛季最高分构造这三个字段。领奖弹窗直接使用响应中的对应字段。

修复：两个接口共用同一构造函数，分别返回前周、上周和当前赛季最高分及对应评级。无成绩时不虚构评级。测试覆盖接口返回一致性及三个时期不同分数/评级的情况。

## 已核对的结算保护

- 时间使用固定 UTC−8，周一 00:00 切换；本次切换点是 `2026-09-21 08:00:00 UTC`，不是夏令时 UTC−7。
- 上周版本为 `1789372800000`，本周版本为 `1789977600000`。
- 只领取最近一个已结束周；奖励配置按该周结束前的时间解析，不会采用新周刚启用的配置。
- 无奖励时不写已领取；领取成功后只标记已结束周。
- `UpdateUsers` 按用户串行加锁，奖励库存与领取标记在同一 SQLite 事务中保存；失败会回滚。
- 现有 Diff 拦截器会同步库存与领取状态变化。新增成功日志在事务提交后记录玩家 ID、周版本和奖励条目数。

## 验证与上线后核对

修复前，新测试复现了空状态列表、累加多发和基础档漏发。修复后运行：

```sh
go test ./internal/service ./internal/gametime ./internal/userdata ./internal/store/sqlite ./internal/interceptor -count=1
```

前四个包全部通过；`interceptor` 包构建通过，无测试文件。`git diff --check` 通过。

部署此修复后，玩家 2 重新登录会补齐已有周成绩的待领取状态。若上周有正分且领取记录未被提前标记，客户端应发起领奖；再核对成功日志中的 `userId=2 weeklyVersion=1789372800000` 和实际库存变化。当前领取窗口在 `2026-09-28 00:00 PST` 结束，超过窗口的历史漏领奖不会由本次修复自动补发。

本次没有核算历史补偿，也未验证“整周未挑战但保留更早赛季最高分”是否应自动获得周奖励；当前服务端仅依据已存在的周成绩结算。

可在数据库只读连接中执行以下查询确认玩家 2 的关键记录：

```sql
SELECT user_id, player_id FROM users WHERE player_id = 2;

SELECT big_hunt_weekly_version, attribute_type, max_score, latest_version
FROM user_big_hunt_weekly_max_scores
WHERE user_id = (SELECT user_id FROM users WHERE player_id = 2)
  AND big_hunt_weekly_version IN (1789372800000, 1789977600000)
ORDER BY big_hunt_weekly_version, attribute_type;

SELECT big_hunt_weekly_version, is_received_weekly_reward, latest_version
FROM user_big_hunt_weekly_statuses
WHERE user_id = (SELECT user_id FROM users WHERE player_id = 2)
ORDER BY big_hunt_weekly_version DESC;
```
