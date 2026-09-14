# CIGC 预售系统

套餐订单 + 推荐/双轨 + 静态释放 + 三类动态奖励。动态与静态入账均为一半 U、一半 ispay。

工程骨架取自金牛（jinniu-new2）：Kratos v3 分层、钱包登录、账本流水、日结占位、管理 JWT。领域规则按预售需求重写。

技术栈：Go 1.27.1、Kratos v3、MySQL 26.7.0（Docker）、部署 Docker Compose（mysql + api + nginx）。

## 已确认对齐金牛的口径

**金额精度**:
业务金额使用 `DECIMAL(36,8)` 与 `shopspring/decimal`，入账与比较前 `Round(8)`。接口展示用 `Display`：整数不带小数点，有小数才保留到有效位。不采用需求文档的 4 位截断。
_Avoid_: Truncate(4)；二进制 float64 算余额；先截断链上原始数量再判断是否足额

**日结**:
进程内 cron（默认 `0 0 * * *`，时区 Asia/Shanghai）。`settle_runs.settle_date` 唯一。普通跑先 `INSERT` 占位，冲突则跳过；占位成功中途失败不自动重跑。`force=1` 仅当 `allow_force_settle=true`。普通结算日取**当前上海自然日**。管理端「测试日结算」`force=1` 结算**下一日**（`max(今日+1, 最近结算日+1)`），用于连点模拟多日静态释放。
_Avoid_: 先查再写防重；生产开 force；无定时只靠手动；按 UTC 自然日占位

**创世地址**:
`CIGC_GENESIS_ADDRESS=0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1`。该地址免推荐码注册；不配则无法产生第一个用户。
_Avoid_: 开发环境放开无邀请注册；把创世地址写进代码

**日结 P0**:
cron `0 0 * * *` Asia/Shanghai；管理端 `POST /settle`、`GET /settle_status`（`/api/admin_cigc`，nginx 把 `/api/admin_dhb/` 转到同一套接口）。时钟接缝可注入，测试钉死上海自然日。批次顺序：占位 → **补漏直推/对碰/管理** → **静态释放** → 按**最大已支付单**回写 `cap_effective` → **按当日封顶解冻剩余冻结**。直推、对碰、管理奖在订单标 paid 时**秒结**；新单抬升的封顶支付当下生效。日结只补漏（防重不双发）。静态仍按日。动态/静态入账均按当天 ispay 现价拆成一半 U + 一半 ispay；未激活进冻结，已激活进可提现。逐用户/逐单推进，中途失败当天不自动重跑（需 `force=1`）。
_Avoid_: 同日无 force 重跑；生产开 force

**直推奖励 (M9)**:
订单标 paid 时秒结推荐人 `Round(8)(amount × direct_rate)`，费率读 `business_configs.direct_rate`（默认 0.10）。再按当天 ispay 现价 **一半 U + 一半 ispay** 入账。已激活进 `available` + `ispay_balance`；未激活进 `lock_balance` + `lock_ispay`（流水 `direct` / `direct_ispay`）。无推荐人（创世）跳过；推荐人账号锁定仍发。同一 `order_id` 只发一次（`ExistsByOrderAndType`）。日结补漏，force 不双发。
_Avoid_: 用安置关系算直推；受对碰封顶限制；动态整额只进 U；等日结才发直推

**对碰奖励 (M11)**:
订单标 paid 时秒结：该单金额沿**安置祖先**加入左右区结余（`user_match_balances`；自己买单不进自己的区），立刻对碰。`pair=min(left,right)`，`credit=min(Round8(pair×match_rate), 当日已生效 cap_effective)`，`match_rate` 默认 0.10。两边都减 `pair`（封顶只砍钱）。`credit` 按当天现价拆一半 U + 一半 ispay（`match` / `match_ispay`）。同一用户同一天可多次对碰。管理奖基数用拆分前的 `credit`（流水 U 金额 ×2）。业绩按 `match_order_applied` 一单一加。锁定仍发。无安置链或一边为 0 则跳过。日结补漏未入账业绩并再对碰一次结余。
_Avoid_: 用推荐树算对碰；force 重复累加当日订单；封顶砍钱就不扣业绩；等日结才对碰

**管理奖 (M12)**:
对碰入账后立刻秒结。基数是该笔**对碰入账** `credit`（不是订单额）。总池 `Round8(credit × manage_rate)`，`manage_rate` 默认 0.30，按推荐链向上找**三代已激活**用户均分（`ManageShares`，余数贴最近一代）。未激活跳过并继续往上找，直到凑齐 3 个已激活或链走完；找不全三代则只发找到的份额，不补给别人。不受 `cap_effective` 限制。账号锁定仍发。份额拆一半 U + 一半 ispay（`manage` / `manage_ispay`）。`remark=manage gen=N source=用户ID key=order|leftover`，按受益人+结算日+remark 防重。日结补漏。
_Avoid_: 用安置树算管理奖；未激活也占一代名额；缺代并入近代；管理奖吃对碰封顶；等日结才发管理奖

**安置树查询 (M13)**:
用户 `GET /api/app_server/recommend_list?address=`：返回安置**直接子**（左右最多 2），`amount` 为该子整棵安置子树累计 `paid_amount`（含自己）。无 address 查自己；查他人仅当对方在自己子树内，否则空列表。管理 `GET .../recommend_list?address=` 同形状，可查任意地址（无地址空列表）。`user_info` 的 `min`/`max`/`total` = 左右结余较小/较大/之和；无结余行则 0。
_Avoid_: 用推荐直推当树；把结余和子树累计 paid 混用；改 level/静态等金牛残留字段

**链上核销 (M14)**:
BSC USDT（默认 `0x55d3…7955`）转入配置的**多收款地址**（`receive_addresses`，百分比合计 100；可用 `CIGC_RECEIVE_ADDRESSES` 覆盖）。默认 4 地址：89% / 5% / 5% / 1%。确认数默认 12。`tx_hash+log_index` 唯一。注册用户转入任一收款地址 → `chain_deposits` `matched` 并**累加 `recharge_balance`**（BuySomething 拆成多笔份额则多笔记入、合计即充值额）。未知发送方 → `abnormal`，不入账。**不再按 pending 订单金额核销**。用户 `GET /deposit_list` 看自己的充值记录；管理 `GET /record_list` 看全部。`user_info` 的 `amountUsdt` / `rechargeBalance` 为充值可用余额；`usdt` 仍是可提现奖励。dapp 充值页调 `BuySomething.buy`，商店下单只扣充值余额。`deposit_cron` 空则只靠管理 `POST /deposit_scan`。收款地址按配置原文写入，不改位数。已有库跑 `scripts/migrate_chain_deposit.sql`、`scripts/migrate_recharge.sql`。
_Avoid_: 把充值记进可提现；未确认块入账；无收款地址仍开扫链；用扫链结果直接标订单 paid

**静态释放 (ispay)**:
下单必选释放档：300 天买价 1200U、600 天 1000U、750 天 800U。购币数 = `订单额 / 档位买价`；日释放币 = `购币数 / 天数`；日产值 = `日释放币 × 当天交易所现价`。再拆一半 U、一半 ispay（ispay = 半额 U / 现价）。例：12000U + 300 天 → 10 币，日释放 0.03333333 币；现价 2000 时日产值 66.66666U → 入账 33.33333U + 0.01666667 ispay。750 天 → 15 币、日 0.02 币、现价 2000 时 20U + 0.01 ispay。已支付且 `paid_at` 落在结算日或之前、已释放天数 < 档位天数才发。按订单+结算日防重。交易所现价接口 `GET /api/app_server/ispay_price` 先保留，测试固定 **2000U**。已有库跑 `scripts/migrate_ispay_static.sql`。
_Avoid_: 未选档位下单；未支付就释放；超过档位天数继续发；把动态整额只进 U

**封顶生效**:
按该用户**已支付订单的最大单金额**向下匹配档位（不累加 `paid_amount`）。取金额 ≤ 最大单的最高档（含已下架套餐），`cap_effective = daily_cap`；不够一档则为 0。新单标 paid 当下若本单档位更高则立刻抬升，对碰秒结立即用新封顶。日结再全量回写作补漏。例：最大单 0/999→0，1000→600，三笔 1000（累计 3000）仍是 600，单笔 4000→1800。
_Avoid_: 只匹配 enabled 套餐；用 paid_amount 累加升档；按各笔订单封顶相加；更大单还等日结才升档

## 预售领域（相对金牛已删除）

不做：认购订单 / 出局倍数 / 提取掉头 / V 级级差 / 平级 / 按静态发代数 / 用可提现奖励买套餐。买套餐只扣 `recharge_balance`。

**套餐**:
金额同时决定可匹配的对碰封顶档位（向下匹配**最大已支付单**）。每档有默认释放天数 `release_days`（300/600/750，缺省 300）。管理 `GET /package_list` 看全部（含下架）；`POST /package_create`、`POST /package_update`（名称/描述/金额/日封顶/天数/排序/上下架）、`POST /package_delete`（已有订单则拒绝，请下架）。金额不可与其他档重复。用户端 Web3 商城只展示 `enabled` 商品。已有库跑 `scripts/migrate_package_days.sql`。
_Avoid_: 自填任意金额；把封顶当成每日固定收益；用多单金额相加升档；把已成交订单的天数改掉

**订单**:
用户 `POST /buy` 校验套餐与释放档后，从 `recharge_balance` 扣款并立即标 `paid`（不足则拒绝）。管理端 `order_pay` 仍可补标旧 pending，不扣充值余额。仅已支付计入 `paid_amount` 与后续业绩。`GET /order_list` 每笔带结算日期、今日/已释放/待释放（USDT+ispay）。结算日期=该单最近一次静态 `settle_date`；今日=该单当日静态（已入账用流水，未入账用当日公式）；已释放=该单历史 `static`+`static_ispay`；待释放=`购币数×现价−已释放产值` 再对半。
_Avoid_: 未支付计入累计；退款接口；买套餐再走一遍链上付款

**内部账户**:
`available_balance` 可提 U；`recharge_balance` 充值页入账、仅用于买套餐；`ispay_balance` 可提 ispay；`lock_balance` / `lock_ispay` 未激活冻结收益；`frozen_balance` **仅** USDT 提现冻结；`frozen_ispay` **仅** ISPAY 提现冻结。实际到账指内部入账。
_Avoid_: 把充值余额和可提现奖励混用；把未激活收益和提现冻结混在 frozen_balance / frozen_ispay

**激活与冻结收益**:
无已支付订单为未激活用户。未激活期间直推/对碰/管理/静态一律进 `lock_balance` + `lock_ispay`。订单标 paid 当下：按**该笔订单封顶**解冻 `min(冻结, 本单封顶)`，超额留在冻结；ispay 按同比例。之后新收益直接进可提现。每个上海自然日 0:00 日结在回写封顶后，再按当日 `cap_effective` 解冻一档剩余冻结（仍超过封顶则只转封顶额度，下一日结继续）。同日 force 不解冻第二档。管理标记支付与链上核销都会走支付当下解冻。
_Avoid_: 未激活把收益记入可提现；激活后仍把新收益进 lock；激活时无视封顶一次解冻全部；日结同一天重复解冻多档

**推荐关系**:
`inviter_id` 绑定后不可改；`user_recommends.path` 物化祖先链。管理奖沿 `inviter_id` 向上找已激活用户。
_Avoid_: 用安置关系算直推/管理奖

**安置**:
表 `user_placements`（`user_id` 唯一；`uk(sponsor_id,side)` 每人左右各最多一个直接子）。与推荐关系分离。创世不落位。注册即按邀请时间自动落位。管理端仍可 `POST/GET /placement`。管理 `GET /downline`（`user_id` 或 `address`）正式模式返回当前人一层：邀请直推名单 + 左右安置，不递归整树。测试 `full_downline=true`（`CIGC_FULL_DOWNLINE`）时展开全部邀请后代与双轨安置；正式上线改回 `false`。用户 `recommend_list` 在全量模式下同样带嵌套 `children`。已有库执行 `scripts/migrate_user_placements.sql`。

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
对碰入账与冻结解冻的每日上限。按最大已支付单匹配，不是累计 paid_amount。支付当下抬升。
_Avoid_: 限制直推或管理奖；用多单金额相加升档

**账本查询 (reward_list / M5)**:
用户 `GET /api/app_server/reward_list`：默认（空/`reqType=1`）返回 `direct|match|manage|static`；`2→static`、`3→direct`、`4→match`、`5→manage`；未知码空列表。分页每页 10，`amount` 为入账 U（一半），`amountTwo` 为对应 ispay。管理 `GET .../reward_list` 返回 `{rewards,count}`，另带 `remark`/`detail`/`category`/`amountTwo`（配对 ISPAY）/`address`/`balanceName`/`orderId`/`orderNo`/`orderAmount`/`orderTitle`/`orderSource`/`sourceAddress`/`settleDate`。`orderSource` 为订单编号与金额组合（如 `C000016 / 50000`）；`sourceAddress` 为来源订单买家地址，管理奖无订单时取 `remark source=` 用户地址。认购单有 `order_no`（`C` + 6 位 ID）。`reason`：空=全类型；`reward`=静态+动态 U；`static`/`dynamic`/`direct`/`match`/`manage`。金额 `Round(8)` 字符串。对碰未产生流水时 `match` 合法空列表。
_Avoid_: 默认混入提现/冻结；改动 dapp 字段名

**管理端加厚 (M6)**:
`GET buy_list` → `orders` 分页（默认全状态，可筛 address/status），`{rewards,count}`，`orderNo` 为认购编号，`one=title_snapshot`，收货字段空。`GET user_list` → 会员分页，兼容旧列填 0/空，真值在 paid/available/`rechargeBalance`（`amountUsdtTwo`）/lock（账号锁定）/lockBalance/lockIspay/activated/cap/frozen/frozenIspay/ispay/`unlockToday`/`unlockTodayIspay`/`todayLockReleased`。后台用户列表展示收益冻结（`lockBalance`/`lockIspay`）；不展示提现冻结列。`GET record_list` → 全部 `chain_deposits`。`POST lock_user`（`user_id`+`lock`）与 `POST unlock_user`；`one=1` 线锁返回 UNSUPPORTED。`sub_money` 固定 `UNSUPPORTED_OPERATION`（不映射 order_pay）。标记支付只用 `order_pay`。
_Avoid_: 退款/删单；把 sub_money 当支付

**管理端调账**:
按地址加减内部账户，正数增加、负数减少，结果不得低于 0。可调：`available`（可提 U）、`recharge`（充值可用）、`lock`（冻结 U）、`ispay`、`lock_ispay`。不动 `frozen_balance`（提现冻结）。兼容入口：`POST /add_money_two`（usdt→available）、`POST /add_money_three`（usdt→recharge）、`POST /set_ispay`（amount→ispay）、`POST /add_lock`、`POST /add_lock_ispay`；统一入口 `POST /adjust_balance`（`address`+`kind`+`amount`）。写流水 `admin_adjust`。金额 `Round(8)`。
_Avoid_: 把调账当成设置绝对值；调提现冻结；用 sub_money 改余额

**管理配置 (M16)**:
`GET /api/admin_cigc/config` 返回 `{config:[{id,key,name,value}]}`，对齐 dapp-admin `Gai.config`。`POST /config_update`（`id`+`value`）只改已有行。可改键：`direct_rate`/`match_rate`/`manage_rate`（0–1）、`min_withdraw_amount`（>0）、`withdraw_fee_rate`（0–1，USDT 提现手续费）、`withdraw_daily_limit`（≥0，USDT 每日上限，0 不限制）、`withdraw_daily_limit_ispay`（≥0，ISPAY 每日上限，0 不限制）、`ispay_price`（>0，测试现价）。数值 `Round(8)`。日结与 `GET /ispay_price` 读库中的 `ispay_price`，缺省 2000。已有库跑 `scripts/migrate_ispay_price.sql`、`scripts/migrate_withdraw_fee.sql`、`scripts/migrate_withdraw_daily.sql`。
_Avoid_: 乱删配置键；费率大于 1；把现价写死在代码里改测试值

**提现状态机 (M8)**:
用户 `POST /withdraw`：USDT / ISPAY 走同一接口（`coinType` 空/0/1/usdt 为 U；2/3/ispay/newispay 为 ISPAY）。未激活（无已支付订单）无论币种都拒绝，提示先购买激活。已激活 USDT：`available → frozen`；已激活 ISPAY：`ispay → frozen_ispay`。流水 `withdraw_freeze` 两条，状态 `pending`，单上记 `asset`。USDT 最低额读 `min_withdraw_amount`（默认 10）；ISPAY 只要金额 > 0。USDT 手续费读 `withdraw_fee_rate`（默认 0.10，0–1，管理可改）：`fee=Round(amount×rate)`，`credited=amount-fee`，到账必须 > 0；冻结仍是申请额，打款只打 `credited`。ISPAY 手续费本刀 0。USDT 每日上限读 `withdraw_daily_limit`（默认 1000，0 不限制），ISPAY 读 `withdraw_daily_limit_ispay`（默认 1000，0 不限制）；两币分开按 Asia/Shanghai 自然日累计申请额；`pending|rewarded|doing|pass` 计入，`rejected|cancelled` 不计。`user_info` 的 `withdrawRate`/`withdrawMin`/`withdrawDaily`/`withdrawToday`/`withdrawRemain` 对应 USDT；`withdrawDailyTwo`/`withdrawTodayTwo`/`withdrawRemainTwo` 对应 ISPAY；`withdrawRateTwo`/`withdrawMinTwo` 为 0。`GET withdraw_list` 分页 `{count,list}` 字段 `id,amount,feeAmount,relAmount,asset,type,status,createdAt`（`pass` 已打款，`cancelled` 已取消）。用户 `POST /withdraw_cancel`（`id`）只能取消本人 `pending`，解冻口径与拒绝相同。管理 `GET withdraw_list` 为 `{withdraw,count}`，另带 `txHash`/`payoutError`/`feeAmount`/`relAmount`，可筛 address、`status`（pending/rewarded/doing/pass/rejected/cancelled）、`withDrawType`（USDT / RAW_NEW）；管理端列表默认筛 `pending`。`POST withdraw_pass`：`pending→rewarded`，对应冻结保持（通过不等于打款）。`POST withdraw_reject`：`pending→rejected`，USDT 解冻回 available，ISPAY 解冻回 ispay。已有库跑 `scripts/migrate_ispay_withdraw.sql`、`scripts/migrate_withdraw_fee.sql`、`scripts/migrate_withdraw_daily.sql`。
_Avoid_: 未激活用户提现；把 ISPAY `coinType` 扣成 U；申请即扣光冻结；审核通过就打链上；pass 时把钱退回可提现；取消已审核单；ISPAY 收手续费；把拒绝/取消单算进当日额度

**USDT 打款**:
热钱包私钥只走环境变量 `CIGC_HOT_WALLET_KEY`，不入库、不写 yaml 真值。默认关：`CIGC_PAYOUT_ENABLED=false`。开启时必须同时有 `CIGC_PAYOUT_MAX_USDT`、热钱包、`CIGC_BSC_RPC`、`CIGC_USDT_ADDRESS`。本刀只打 BSC USDT（18 位）；ISPAY `rewarded` 跳过并写 `payout_error=本刀只打 USDT`。ISPAY 链上打款已搁置：等你给出合约地址、精度、转账 ABI 后再做，未给合约前不写 ISPAY transfer、不烧 `frozen_ispay`。触发：管理 `POST /withdraw_payout`（可选 `id` 打一笔，否则批量最多 20）+ `payout_cron`（空则不定时）。状态：`rewarded→doing`（CAS 后再发交易）；成功回执后 `doing→pass`，`SubFrozenBalance` + 流水 `withdraw`（冻结负数，备注 `payout <hash>`）；发交易失败或回执 revert：`doing→rewarded`，冻结保持，可重试；`doing` 已有 hash 且回执未出：保持 doing；`doing` 无 hash：重置回 rewarded。超过单笔上限只跳过不发。已有库跑 `scripts/migrate_payout.sql`。
_Avoid_: 审核通过自动打款；ISPAY 打链上；无 CAS 重复发交易；回执未确认就烧冻结；把热钱包写进库或 yaml

**工程补齐**:
Compose `api` 注入扫链/收款/日结/打款环境变量；空 `CIGC_RECEIVE_ADDRESSES` 不覆盖 yaml。已有库 `make migrate-existing`（`scripts/migrate_existing.sh`，可重复，含 `migrate_ispay_withdraw.sql`、`migrate_payout.sql`、`migrate_withdraw_fee.sql`、`migrate_withdraw_daily.sql`、`migrate_recharge.sql`）。冒烟 `make smoke`：`/health`、`package_list`、`ispay_price`、管理登录/config/`settle_status`。正式 ISPAY 行情与 ISPAY 打款仍未做。
_Avoid_: 默认打开打款；接外部行情

**管理端首页**:
`GET /api/admin_cigc/all`（及 admin_dhb）返回看板：`totalUserR`/`totalUser`/`todayUserR`/`todayUser`、`buyTotal`/`todayBuy`、`balanceUsdt`、`todayOne`（静态）/`todayTwo`（动态）/`todayThree`、`totalReward`、`todayWithdraw`/`totalWithdraw`、`totalIspay`。日切 Asia/Shanghai。金额 `Round(8)` 字符串。
_Avoid_: 把 ispay 半额算进 U 奖励；把已拒绝/已取消提现算进提现合计

**管理菜单占位**:
`GET /api/admin_dhb/my_auth_list`（及 admin_cigc）返回 `{super:"1",auth:[]}`，让现有 dapp-admin 登录后能挂全量菜单。不做权限模型。
_Avoid_: 在未给规则前做多管理员/菜单权限

**用户端下单对齐**:
dapp 充值页链上转入后等扫链入 `recharge_balance`；商店读 `package_list`，下单必须选 300/600/750，只用充值可用余额，预览购币/日释放/一半 U+一半 ispay。`user_info` 带 `amountUsdt`/`rechargeBalance`/`ispay`/`ispayAmount`/`ispayPrice`/`release_tiers`/`lock`/`lockIspay`/`activated`/`capEffective`/`unlockToday`/`unlockTodayIspay`/`todayLockReleased`/`frozen`/`frozenIspay`/`withdrawRate`/`withdrawMin`/`withdrawDaily`/`withdrawToday`/`withdrawRemain`/`withdrawDailyTwo`/`withdrawTodayTwo`/`withdrawRemainTwo`。资产页展示可提现、**收益冻结**（`lock`/`lockIspay`）、**提现冻结**（`frozen`/`frozenIspay`）、日封顶、当日还能解冻多少（未激活或今日已日结解冻为 0；否则 `min(收益冻结, cap_effective)`，ispay 同比例）。订单页读 `order_list`。ISPAY 提现走同一 `POST /withdraw` + `coinType=3`。待审核可 `POST /withdraw_cancel` 取消。
_Avoid_: 任意金额下单；不选释放档；把 ispay 显示成 USDT `raw`；把收益冻结/提现冻结混成可提现；把当日可解冻当成可提现
