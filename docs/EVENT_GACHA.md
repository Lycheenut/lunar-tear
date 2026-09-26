# Event Gacha ticket pools

Event rewards are read from the existing `--gacha-config` JSON file at startup
and on master-data reload. Each supported ticket tier is an independent entry
in `eventBanners`; it owns its boxes, prices, inventory counters and reset state.
The catalog derives the activity window and unlock conditions from the linked
event chapter. Ticket families are reconstructed from the installed ticket and
Gacha title assets, including older events that only have bronze and silver.

For example, Aurora Dynast uses:

| Tier | Gacha ID / `eventBanners` key | Ticket ID | Banner asset |
| --- | --- | --- | --- |
| Bronze | `329001` | `6055` | `event_329_01` |
| Silver | `329011` | `6056` | `event_329_11` |
| Gold | `329021` | `6057` | `event_329_21` |

Existing lowest-tier IDs and configuration remain valid. Silver and gold are
hidden until their own reward boxes are configured; rewards are never copied
implicitly from bronze. Each pool has single and ten-ticket draw options and
accepts only its own ticket. Rewards may themselves grant another tier's ticket.

Each `eventBanners` value has the same shape (illustrative reward only):

```json
{
  "boxes": [{
    "groupWeights": { "limited": 10000, "unlimited": 0 },
    "limitedRewards": [{
      "possessionType": 6,
      "possessionId": 6056,
      "count": 1,
      "maxCount": 1,
      "featured": true,
      "jackpot": true
    }],
    "unlimitedRewards": []
  }]
}
```

Weights must total 10000. Limited rewards need positive stock; unlimited rewards
need positive weights and no stock. Every event box needs a jackpot. Non-final
boxes advance after all jackpots are drawn; the final box resets only after all
limited rewards are drawn. Draws, promotion counts and web rates use the same
current box. Invalid rewards or unknown pool IDs reject the configuration.

`eventSchedules` on the lowest-tier ID applies to all tiers. An explicit schedule
on a silver/gold ID overrides that inherited window. Existing configuration
version 1 is retained.

On `prod`, the Gacha configuration tool lists every tier, including unconfigured
pools. The ticket-tier filter and each pool's ticket ID/name distinguish pools
from the same event. Adding/removing boxes or editing rewards affects only the
selected pool; publishing saves all tiers together using the existing config
hash check and atomic hot reload. Activity groups retain one member for the
base event so existing activity schedules continue to cover all three tiers.
