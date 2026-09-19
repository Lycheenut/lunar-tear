# 非主线 Companion 赠礼

玩家首次解锁登录奖励功能时，与２Ｂ（破ノ攻機）及其相关奖励一起，直接获得
ID **31–53，共23个** Companion。覆盖已核对的全部非主线关卡直接奖励
Companion，包含 Pod 042，不采用仅排除 Record／Variation 的4个名单。

- **49、50、51：50级**。这三个无法正常强化。
- 其余20个：1级，包括53。
- 全部直接进入 Companion 背包，通过客户端 `IUserCompanion` 差量同步，
  不经过礼物箱，不受历史活动时间限制。

## 提前持有不会解锁编队槽

客户端 `CalculatorUnlockCondition.IsUnlockOrganizationCompanion`
（ARM64 RVA `0x29D018C`）只读取当前用户的 Companion 教程进度：
`TutorialType = 8`，`ProgressPhase >= TutorialPhases.CompanionFirst`（10）。
未找到教程记录时返回未解锁，判断不读取 Companion 持有数量。

核对依据：

- `0x29D021C` 设置教程类型8；`0x29D0244` 调用
  `EntityIUserTutorialProgressTable.TryFindByUserIdAndTutorialType`（`0x35DB7D0`）。
- `0x29D025C` 读取 `ProgressPhase`，随后与静态字段 `CompanionFirst`
  （字段偏移 `0x18`）比较。`TutorialPhases` 静态初始化函数（`0x2DAA7DC`）
  在 `0x2DAA864` 将该字段设为10。
- `DeckActorView.SetUnlock` 和 `OrganizationDeckActorView.SetUnlock`
  分别在 `0x302BF6C`、`0x313E1C0` 调用此判断；自动编队候选列表也使用该判断。
- 本地原始客户端、当前客户端及英文构建中的这段解锁函数字节一致。

因此可以在 Companion 教程之前发放。这次赠礼不修改 Companion 教程进度，
编队槽仍按原教程条件解锁。

## 发放与老账号覆盖

发放时机与[登录奖励解锁赠礼](LOGIN_BONUS_UNLOCK_REWARD.md)完全一致：
`TutorialType = 3`（MenuSecond）的进度首次从小于20跨越到大于等于20。
两种教程进度接口都通过 `store.GrantLoginBonusUnlockReward` 发放整套奖励，
教程进度和背包变化在同一事务中提交；并发、重试、登录及领取签到不会重发。
无需已经拥有 Companion，也不新增领取标记或数据库迁移。

后续 Companion 教程四选一仍只发原有所选奖励，不再触发这23个赠礼：

| ChoiceId | 原有四选一奖励 CompanionId |
| --- | --- |
| 1 | 2 |
| 2 | 1 |
| 3 | 7 |
| 4 | 10 |

老账号由 `prod` 分支的 `server/cmd/repair-player-data` 一次性补发，资格同样
为登录奖励已解锁，不再根据是否已有 Companion 判断。
新旧路径共用 `store.GrantCompanionUnlockReward`：只补缺失项，保留已有
Companion 的 UUID、获得时间和等级；49–51不足50级时原地修正到50级，
不产生重复 Companion 补偿。工具默认只读预览，并报告数量和等级变化。

验证覆盖：两种教程接口的解锁边界、跳过教程、重复及并发请求、四选一兼容性、
数据库持久化、客户端差量、Companion 教程不变、修复预览与执行一致、
已有等级保留及整批事务回滚。
