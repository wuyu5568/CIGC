# CIGC 预售系统

套餐订单 + 推荐/双轨 + 三类奖励入内部账户。不做独立挖矿/静态收益模块。

工程骨架取自金牛（jinniu-new2）：Kratos v3 分层、钱包登录、账本流水、日结占位、管理 JWT。领域规则按预售需求重写。

技术栈：Go 1.27.1、Kratos v3、MySQL 26.7.0（Docker）、部署 Docker Compose（mysql + api + nginx）。

## 已确认对齐金牛的口径

**金额精度**:
业务金额使用 `DECIMAL(36,8)` 与 `shopspring/decimal`，入账与比较前 `Round(8)`。不采用需求文档的 4 位截断。
_Avoid_: Truncate(4)；二进制 float64 算余额；先截断链上原始数量再判断是否足额

**日结**:
进程内 cron（默认 `0 0 * * *`，时区 Asia/Shanghai）。`settle_runs.settle_date` 唯一。普通跑先 `INSERT` 占位，冲突则跳过；占位成功中途失败不自动重跑。`force=1` 仅当 `allow_force_settle=true`。结算日取**当前上海自然日**（与金牛相同，不是「结算昨日」）。
_Avoid_: 先查再写防重；生产开 force；无定时只靠手动；按 UTC 自然日占位

**创世地址**:
`CIGC_GENESIS_ADDRESS=0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1`。该地址免推荐码注册；不配则无法产生第一个用户。
_Avoid_: 开发环境放开无邀请注册；把创世地址写进代码

**日结 P0**:
cron `0 0 * * *` Asia/Shanghai；管理端 `POST /settle`、`GET /settle_status`（`/api/admin_cigc`，nginx 把 `/api/admin_dhb/` 转到同一套接口）。时钟接缝可注入，测试钉死上海自然日。批次顺序：占位 → **当日已支付订单直推入账** → **对碰入账** → 按 `paid_amount` 回写 `cap_effective`。不发管理奖。逐用户/逐单推进，中途失败当天不自动重跑（需 `force=1`）。
_Avoid_: 支付后立刻改 cap_effective；同日无 force 重跑；生产开 force

**直推奖励 (M9)**:
日结对 `paid_at` 落在该上海自然日的 `paid` 订单：奖励推荐人 `Round(8)(amount × direct_rate)`，费率读 `business_configs.direct_rate`（默认 0.10）。入 `available_balance` + `ledger`（`direct`，带 `order_id`+`settle_date`）。无推荐人（创世）跳过；推荐人锁定仍发。同一 `order_id` 只发一次（`ExistsByOrderAndType`），force 重跑不双发。
_Avoid_: 支付瞬间发直推；用安置关系算直推；受对碰封顶限制

**对碰奖励 (M11)**:
日结在直推之后、刷新封顶之前。当日新 `paid` 订单金额沿**安置祖先**加入对应左右区结余（`user_match_balances`；自己买单不进自己的区）。`pair=min(left,right)`，`credit=min(Round8(pair×match_rate), 当日已生效 cap_effective)`，`match_rate` 默认 0.10。两边都减 `pair`（封顶只砍钱）。入 `available` + `ledger.match`（按用户+结算日防重，force 不双发；业绩按 `match_order_applied` 一单一加）。锁定仍发。无安置链或一边为 0 则跳过。不回溯历史支付。不做管理奖。
_Avoid_: 用推荐树算对碰；支付瞬间对碰；force 重复累加当日订单；封顶砍钱就不扣业绩

**日结后刷新封顶**:
批次先使用用户当前 `cap_effective` 计算对碰，再按最新 `paid_amount` 向下匹配档位写回 `cap_effective`。取金额 ≤ `paid_amount` 的最高档（含已下架套餐），`cap_effective = daily_cap`；不够一档则为 0。锁定用户也回写。例：0/999→0，1000→600，4000→1800，160000→100000。当日新购买提高的封顶从下一次日结起生效。
_Avoid_: 只匹配 enabled 套餐；按各笔订单封顶相加

## 预售领域（相对金牛已删除）

不做：认购订单 / 出局倍数 / 静态利率 / 提取掉头 / V 级级差 / 平级 / 按静态发代数 / 账户余额再认购。

**套餐**:
十档固定金额购买，金额同时决定可匹配的对碰封顶档位（向下匹配累计已支付）。
_Avoid_: 自填任意金额；把封顶当成每日固定收益

**订单**:
创建后为 pending；仅已支付计入 `paid_amount` 与后续业绩。P0 由管理端标记已支付；链上核销后续接入。
_Avoid_: 未支付计入累计；退款接口

**内部账户**:
`available_balance` 可提奖励；`frozen_balance` 提现冻结。实际到账指内部入账。
_Avoid_: 与金牛「账户余额认购」混用

**推荐关系**:
`inviter_id` 绑定后不可改；`user_recommends.path` 物化祖先链，供三代管理奖。
_Avoid_: 用安置关系算直推/管理奖

**安置**:
表 `user_placements`（`user_id` 唯一；`uk(sponsor_id,side)` 每人左右各最多一个直接子）。与推荐关系分离。创世不落位。注册即按邀请时间自动落位。管理端仍可 `POST/GET /placement`。已有库执行 `scripts/migrate_user_placements.sql`。

1. 同一邀请人按邀请先后：第 1 人挂其左区，第 2 人挂其右区。
2. 被邀请人的左区是**共享链**：邀请人和该被邀请人都能往这个左区挂人。
3. 第 3 人起只在「左区首位、右区首位」两条左链上找空位：**先浅后深，同层从左到右**。某格已有人（不论谁请的）就看同层右侧，再看下一层左侧。
4. 例：B 先占自己左区（挂了 D），C 左空 → A 下一人挂 C 左；再下一人挂 D 左。
5. 例：C 先占自己左区，B 左空 → A 下一人挂 B 左（称 E）；再下一人挂 **E 左**（不先挂 C 那条链更深处）。
6. 例：D 在 B 左、E 在 C 左、D 又请了 R 占 D 左 → A 下一人先看 E 左，空则挂 E 左；E 左也有人才挂 R 左。
_Avoid_: 被邀请人先占左区就对邀请人关链；同层未看完右侧就钻左链更深；层序填满每人的右区；把推荐人当成安置上级；用安置关系算直推

## Language

**已支付累计金额 (paid_amount)**:
会员全部已支付订单金额之和。
_Avoid_: 未支付订单；把封顶额度相加

**有效封顶 (cap_effective)**:
日结对碰使用的每日上限；仅限制对碰入账。
_Avoid_: 限制直推或管理奖；支付后未日结就当已生效

**账本查询 (reward_list / M5)**:
用户 `GET /api/app_server/reward_list`：默认（空/`reqType=1`）只返回 `direct|match|manage`；`3→direct`、`4→match`、`5→manage`；未知码空列表。分页每页 10，字段 `id,amount,amountTwo,reward,name,address,num,reason,createdAt`（`address/num` 暂空）。管理 `GET .../reward_list` 返回 `{rewards,count}`，另带 `remark`，可筛 `address`+`reason`。金额 `Round(8)` 字符串。对碰未产生流水时 `match` 合法空列表。
_Avoid_: 默认混入提现/冻结；改动 dapp 字段名

**管理端加厚 (M6)**:
`GET buy_list` → `orders` 分页（默认全状态，可筛 address/status），`{rewards,count}`，`one=title_snapshot`，收货字段空。`GET user_list` → 会员分页，兼容旧列填 0/空，真值在 paid/available/lock/cap/frozen。`POST lock_user`（`user_id`+`lock`）与 `POST unlock_user`；`one=1` 线锁返回 UNSUPPORTED。`sub_money` 固定 `UNSUPPORTED_OPERATION`（不映射 order_pay）。标记支付只用 `order_pay`。
_Avoid_: 退款/删单；把 sub_money 当支付；改 dapp-admin 仓库

**提现状态机 (M8)**:
用户 `POST /withdraw`：`available → frozen`，流水 `withdraw_freeze` 两条（available 负 / frozen 正），状态 `pending`。最低额读 `min_withdraw_amount`（默认 10）。手续费本刀 0，`credited_amount=amount`。`GET withdraw_list` 分页 `{count,list}` 字段 `id,amount,status,createdAt`。管理 `GET withdraw_list` 为 `{withdraw,count}`，可筛 address/status。`POST withdraw_pass`：`pending→rewarded`，冻结保持。`POST withdraw_reject`：`pending→rejected`，解冻回 available + `withdraw_unfreeze`。不做用户取消、不做链上打款。
_Avoid_: 申请即扣光冻结；pass 时把钱从系统抹掉；映射到 payout
