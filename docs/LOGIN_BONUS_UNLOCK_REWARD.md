# 登录奖励解锁赠礼

登录奖励功能解锁时，一次性直接发放以下物品到背包：

| 类型 | 名称 | ID | 数量 |
| --- | --- | --- | --- |
| 服装 | ２Ｂ（破ノ攻機） | 24008 | 1 |
| 素材 | 戦いの書：破ノ攻機 | 311211 | 40 |
| 素材 | 覚醒石：破ノ攻機 | 313197 | 5 |
| 武器 | 三式戦術刀 | 240271 | 1 |
| 素材 | 天然水晶：軽量 | 312011 | 4 |

数量来自原版 [Japanese 2nd Anniversary 赠礼公告转载](https://www.reddit.com/r/NieRReincarnation/comments/11548g8)。本功能仅发放上述五项。

## 解锁条件

客户端 `CalculatorPromotion.HasLoginBonus`（ARM64 RVA `0x2728CE0`）
调用 `CalculatorOutgame.IsEndMomMenuTutorial`（`0x2721974`）。后者要求
`IUserTutorialProgress` 中 `TutorialType = 3`（MenuSecond）的
`ProgressPhase >= 20`（TutorialPhases.MomMenuEditDeck）。阶段常量由
`TutorialPhases` 的静态初始化函数（`0x2DAA7DC`）确认。

主数据 `m_tutorial_unlock_condition` 将 MenuSecond 关联到场景 42；该场景属于
Quest 12，是第一章 Normal 之后下一章的起始关卡。因此发放条件采用客户端的
教程阶段，而非仅检查第一章最后一关的通关记录。客户端另外检查是否正在关卡中、
是否已播放当日登录奖励，这些是展示时机条件，不是永久解锁条件。

## 发放与老账号覆盖

- 两种提交教程进度的接口仅在进度首次从未解锁跨越到已解锁时发放。教程进度和
  整套背包变动在同一事务中提交；重复、并发或倒退的教程请求不会再次发放。
- 使用已有教程进度判断，不新增任何领取字段或数据库迁移。登录和领取签到
  均不自动补发。消耗材料、出售武器不会改变教程进度，因此也不会触发重发。
- 已符合条件的老账号使用 `prod` 分支的 `server/cmd/repair-player-data` 一次性补发，具体命令
  见该目录的 `README.md`。工具默认只读预览，`--apply` 才提交全部账号的变动。
  工具不写执行标记，重复执行会重复发放；应停服补发后再启动带新逻辑的服务。
- 使用现有物品发放逻辑初始化角色、技能和武器记录；已有同服装时按现有规则转换
  重复服装奖励。奖励直接进入背包，不经过礼物箱，也不受历史周年活动日期限制。

## 礼物箱兼容性

礼物箱支持 `ExpirationDatetime = 0` 的永久礼物，但当前没有“武器附带服装”
这一组合领取类型。服务端 `grantGift` 对每条礼物只调用一次对应物品的 `GrantFull`，
武器和服装的发放彼此独立。拆成两条礼物允许玩家只领取其中一条。

客户端 `CalculatorGift.CreateDataPossessionItem`（`0x2940F38` / `0x2940FD4`）
也只读取单个物品类型、ID 和数量。`GiftCellSetup.AdditionThumbnailData`
（`0x304B8F8`）通过 `CalculatorPossession.GetItemName` 取名称，武器和服装
分别使用 `WeaponName` 与 `CostumeName`，没有附带服装名称的组合分支。
因此保留整套在同一事务中直接进入背包的方式。
