# Dokan 配图上传清单

Dokan 的 `m_dokan_content_group.ImageId` 与客户端资源包按以下规则对应：

- 源资源包：`server/assets/revisions/0/assetbundle/ui/mom_promotion/{language}/banner/prm{ImageId}.assetbundle`
- R2 对象键：`assets/ui/mom_promotion/{language}/banner/prm{ImageId}.png`
- `{language}` 为 `en`、`ja`、`ko`。
- `ImageId` 最少补齐三位，例如 `5 → prm005`，`535 → prm535`，`1005 → prm1005`。

资源包不能直接作为浏览器图片使用。需要从每个资源包导出其配图为 PNG，再上传至对应 R2 对象键。

## 当前有效活动

以仓库 master-data 和 2026-08-11 为基准，当前有效的 16 条 Dokan 共引用 40 个唯一图片 ID；这些 ID 均存在英、日、韩三套资源包，因此共需上传 120 个 PNG：

```text
535, 536, 568, 569, 598, 599, 639, 640, 641, 680,
681, 682, 758, 759, 760, 765, 766, 830, 831, 832,
905, 906, 907, 931, 932, 1005, 1006, 1007, 1039, 1040,
1041, 1048, 1049, 1050, 1111, 1112, 1113, 1116, 1117, 1118
```

例如 `ImageId=535`：

```text
源：server/assets/revisions/0/assetbundle/ui/mom_promotion/en/banner/prm535.assetbundle
目标：assets/ui/mom_promotion/en/banner/prm535.png
```

全部历史 Dokan 共引用 865 个唯一图片 ID。管理页的“配图”列会根据当前选择的语言显示预览；地址返回 404 时，会直接列出该行缺少的 R2 对象键，可按需补传历史资源。
