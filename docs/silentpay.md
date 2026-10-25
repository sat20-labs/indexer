# BaseIndexer SilentPay 实现与接口

日期：2026-10-11。
设计依据：`SAT20_BIP352_Silent_Payments_Design_v1_4.md` 及后续确认的实现取舍。
范围：Bitcoin L1 Indexer。未修改 Wallet SDK、PWA、DID 属性或聪网协议。

## 1. 组织方式与数据来源

SilentPay 是 `BaseIndexer.silentpay` 内部组件，不是新的资产协议索引器，不创建独立数据库，不修改 TxAssets，不识别用户身份，也不接收 scan/spend 私钥。

`FetchBlock` 保留输入的原始 SignatureScript（仅内存）和 witness，并保留原交易 txid。Base 已解析 prevout 后，`processSilentPay` 在现有区块同步路径中预计算公共扫描记录，不经过自己的 HTTP 服务重复拉取当前交易，不依赖 Ordinals 起始高度。

`common/silentpay.go` 实现 BIP352 v0 合格输入提取、Taproot 偶数 Y、NUMS script-path 排除、未来 witness 版本排除、公钥聚合、最小序列化 outpoint 和 `T = input_hash * A`。输出端保持原始 32 字节 witness program，不能因金额、dust、资产承载或尚不属于任何钱包而删掉候选。

主网公共扫描记录从 Taproot 激活高度 709632 起形成连续覆盖；Base 仍从创世区块重建。测试链使用明确网络域，存储、快照和游标不能跨网络混用。

### 已确认的脚本支持缺口

本版不支持 P2PK、裸多签等无法通过现有 Base 地址表示无损还原的源输出类别，包括混用这些来源的交易。不修改 Base 地址存储，不增加历史 prevout RPC 补读。

这里的缺口不是 BIP352 算法可以自动解决的问题：Base 的 `GetAddress()/EncodeAddress()` 可能将原 P2PK/裸多签表示成 P2PKH 地址，`prefetchIndexesFromDB()` 再按地址重建脚本；仅在算法中排除 P2PK，不能还原已经丢失的原始类型。因此不宣称这些交易已被可靠识别或自动跳过，也不宣称内存路径与落库重载路径在该范围内完全一致。注释位于 `indexer/base/silentpay_records.go` 的 `spCandidateRecord()`，紧邻输入 prevout 接入点。

`CheckSelf` 的原始交易校验不因此放宽；若遇到该缺口导致的差异，仍按既有规则报错/panic。未来扩展支持时，应先解决原始脚本语义保留问题。

## 2. 同库数据与持久化

| Base DB 前缀 | 数据 |
| --- | --- |
| `sp/b/<height>` | 区块 hash、按交易序排列的候选目录；零候选区块也写目录 |
| `sp/a/<height>/<tx_index>` | 每候选一份 txid、33 字节 T、全部 P2TR 输出的真实 vout 和 32 字节公钥 |
| `sp/l/<height>/<tx_index>` | 指向同位置 Archive 的未花费位图；每位对应 Archive 输出数组位置，不是真实 vout |
| `sp/schema` | 当前公共扫描编码版本和网络 |
| `sp/commit-pending` | Base 逻辑提交尚未完成时的持久化保护标记 |

高度和交易位置键使用定宽大端整数；值使用版本字节、CompactSize 紧凑计数和位置差分，公钥保持二进制。txid 仅在 Archive 中持久化一次以支持全部花费后的位置定位，正常扫描响应不重复传输它。金额从已有原始交易接口验证，不建立第二套金额/余额数据库。

Live 的作用是筛选交易，不是裁剪交易内已花费输出。`k=0` 已花费、`k=1` 未花费时，仍发送同笔交易全部 P2TR 输出及位图；SDK 必须先匹配前者才能继续扫描后者。最后一个 P2TR 输出花费后删除 Live 行，保留 Archive。

消费已落库 P2TR 输出时，若 Archive 不存在，则用该高度的候选目录确认是否为非候选：只有目录也排除该交易才允许跳过；目录有记录而 Archive 缺失即为数据损坏，停止处理。

### 紧凑内存、按需读取与释放

磁盘 schema 仍为 v1，记录字节和 HTTP JSON/CBOR 字段不变。本次主要消除内存展开和重复保留，而不是再增加一种磁盘格式。

正常区块处理直接从已有 Base 交易生成紧凑 Archive 字节。未落盘的 `spPendingBlock` 只保存 hash、候选交易位置和不可变记录字节，不再长期保留展开的 `SPTransaction` 及逐输出分配的公钥 slice。Clone 共享不可变记录，仅独立复制可变 Live 修改；Subtract 只移除冻结批次所属引用，不删除较新的 Live 变更。落库直接写这些字节，不再次把所有输出组装并序列化一遍。

消费处理只收集当前区块用到的输入引用，按创建交易位置及 vout 排序，然后逐来源处理。每个来源只读取一次对应的紧凑 Archive 行，流式走过 vout 差分，仅比较实际消费输出的公钥、更新相应位图；处理完一个来源后不保留其历史记录，再读取下一个。取消原先整个区块持续持有的展开来源交易缓存。Live 位图每个来源只复制一次，不再每个 input 复制。

现有 KVDB 以完整 value 读取，故这里不是磁盘字节范围读取：仍需读取被引用交易的完整紧凑行，但不会展开或缓存它的全部历史输出对象。没有新增每输出索引、永久缓存或缓存淘汰服务。

Base 已在调用 SP 前按交易顺序解析合法区块的输入。SP 先按原交易序登记本块不可变候选，再按来源归组应用所有消费，包括消费交易不含 P2TR 输出的情况。该分组只排序临时输入引用，不修改 Base 的交易/input 数组；同块父子交易和同块创建后全部花费仍产生相同的最终 Archive/Live 状态。

公开查询需要返回完整输出时，才创建当前页的 `SPTransaction`。公钥和 T 使用当前调用拥有的原始记录 buffer 中的只读切片，不逐个申请 32 字节对象；不暴露数据库 iterator 的临时缓冲区。查询仍受既有分页限制。`txids-by-position` 和 `outpoints/query` 按创建位置（height、tx_index）对请求内最多 500 项临时归组：每个来源只读取一次候选目录和紧凑 Archive，仅在实际对账时读取一次 Live 位图。直接流式遍历差分 vout，不为定位 txid 或少量 vout 构建完整输出对象列表；重复 vout 共享 Base UTXO 查询。完整校验每条紧凑记录，包括未命中位置之后的尾部，损坏仍按原规则失败。回复仍按请求原顺序排列，原有逐项错误/快照锚点与 HTTP 结构不变。归组数据只在当前请求中存在，用完释放，不增加持久化索引或全局缓存。

持久化完成后清空提交实例的 pending maps。普通 RPC 副本只保留 SP 数据库和固定视图池，不持有 pending Archive/Live，避免旧 RPC 实例延长已提交记录的存活时间。释放指的是取消这些缓存引用，由 Go GC 回收；数据库自身缓存及有效快照的版本保留机制不改变。

### 提交和重组

提交顺序：写入并同步 pending 标记；写 Base/Archive/Live/schema/SyncStats 批次；Flush 并同步数据；删除并同步 pending 标记；清理已提交增量。重复空提交时仅在确有未落库新区块的情况下核对 pending 末块 hash；连续高度校验仍保留。即使后端拆分 WriteBatch，也不能将未完成状态发布为新基线。**任何阶段持久化失败：保留可检测的保护状态，然后立即 panic，不能继续同步。** 启动发现 pending 标记、已存在 SP 记录却缺少 schema，或已初始化 SP 的锚点/版本不一致时直接 panic，不自动清除、修复或在线回填。**唯一兼容路径**：已有 Base 已提交高度，但 `sp/schema` 不存在且 `sp/` 前缀完全无数据，视为从未初始化 SP 的旧库；此次运行跳过 SP 处理、落盘及自检，普通 Base 和其他资产索引继续运行。五个 SP RPC 均返回 HTTP 503（`code=-1`、`msg=silentpay_not_initialized`）。重新启动仍保持此状态，不从当前高度自动开始生成不完整 SP 历史。新建 Base DB 继续从头启用 SP；需要在旧库启用完整 SP 时必须另行从头重建。没有新增配置项、持久化标记或在线迁移。

关闭、重组重载和自检退出前释放 SP read views，再关闭 Base DB。重组仍使用现有 Base 持久化安全基线；不新增独立的 SP undo 数据库，不承诺越过 Base 安全范围的任意深度重组可自动恢复。

按已确认取舍，不额外处理“非空数据库意外丢失 SyncStats”的异常场景，不增加新的检测、修复或状态推断逻辑，保持现有初始化行为。

## 3. 固定视图

通过可选数据库能力接口 `common.SnapshotDB` 取得原生只读视图，分别实现 Pebble snapshot 与 Badger read transaction。原 `KVDB` 接口没有新增所有调用者必须实现的方法。

Live 快照固定网络、schema、Base 已提交高度及 hash；每一页使用同一数据库版本。不是普通 RpcIndexer.Clone，也不是每页从最新库读取再贴一个旧高度。

当前资源边界：同一池最多 4 个不同高度的共享快照，每个最长保留 1 小时；到期释放未在用的视图，正在处理的单页通过引用计数完成后释放。相同锚点复用快照。数量满时返回 429，不偷偷淘汰一个仍有效的旧游标。长扫描期间不持有全局写锁；每个 HTTP 查询仍经过 IndexerMgr 的现有读准入保护，避免数据库重载竞态。

游标由带 MAC 的作用域信息构成，绑定 mode、范围、锚点和快照，采用 keyset 分页。到期或重启后旧 token 返回明确错误。正常最新提交不会改变已发出的快照。

这些限制是初始资源保护值，不是主网容量结论；真实移动端恢复耗时、旧版本保留空间及吞吐仍须实测。

## 4. HTTP 接口

以下路径相对于既有网络 basePath，例如 `/btc/mainnet`。全部直接在现有 `rpcserver/base.Service.InitRouter` 注册，handler 统一使用 `(s *Service)`，通过已有 Model 所持的 IndexerMgr 读取 SP 数据；已删除独立 `SilentPayService`、独立 `InitRouter` 和 SP 专用 Gin 中间件。SP 能力接口保持可选，不扩展项目共用的大型 `base_indexer.Indexer` 接口。

### 状态与快照

```text
GET /v1/sp/status
GET /v1/sp/live-snapshot
```

状态包括 network、schema_version、coverage_start_height、base_anchor、data_ready 和 supported_modes。状态查询只读取已提交的本地 Base/SP 数据，不同步调用 bitcoind。

`base_anchor` 是完整持久化的 Base 视图，可能落后 bitcoind tip。`data_ready` 表示该已提交前缀可读，不等于新库已经完成主网上线验收。已删除不能随 HTTP 请求取消的 `transaction_lookup_ready` 实时 bitcoind 探针；实际命中交易是否可读，以已有 `rawtx/batch` 逐项验证，不将本地索引就绪误认为 Bitcoin Core 历史交易可查。

快照返回 snapshot_id、anchor、expires_at、coverage_start_height 和 schema_version。

### 扫描数据

```text
GET /v1/sp/scan-data?mode=live&snapshot_id=...&from_height=...&limit=100&cursor=...
GET /v1/sp/scan-data?mode=archive&from_height=...&to_height=...&anchor_hash=...&limit=100&cursor=...
```

默认 `format=cbor`，Content-Type 为 application/cbor；调试可用 `format=json`。CBOR 中字节串直接是二进制，JSON 中 Go byte slice 使用 base64，不是 hex。字段名在两种格式中相同。

```text
{
  snapshot_id?, anchor: {height, block_hash},
  blocks: [{height, block_hash, transactions: [
    {tx_index, tweak, p2tr_outputs: [{vout, output_key}], unspent_bitmap?}
  ]}],
  next_cursor?, done
}
```

Live 附固定快照位图；Archive 不附最新位图。Archive 返回零候选区块目录；Live 可以省略没有活跃候选的区块。正常扫描不返回 txid、金额或用户余额。

limit 默认 100，最大 500 个候选交易；同时限制页估算体积和每页区块头数量。单笔交易不因分页而裁掉部分输出。编码完成后才发送响应，错误不能伪装成一个完成页面。

Archive 首次响应锚定 to_height/hash；下一页保持同一 to_height、anchor_hash 和范围，不把 to_height 省略后滚动到新 tip。请求结束高度完全早于 SP 覆盖起点时，直接返回 400 `invalid_silentpay_request`，绝不返回当前 Base 高度来充当旧区间的扫描完成锚点。Live 固定 snapshot_id。`done` 是完整枚举结束声明，不是 Bitcoin 共识承诺；网络截断或 error 不等于 done。

### 命中后的定位与交易读取

```text
POST /v1/sp/txids-by-position
[
  {"height": 900000, "block_hash": "...", "tx_index": 1}
]
```

本实现要求同时提供从扫描数据已知的 height、block_hash、tx_index，直接读取同位置 Archive；optional expected_txid 可作调用方一致性约束。返回既有风格的 code/msg/data 及每项 txid 或 error。同一批重复请求复用来源读取，仅从完整校验过的紧凑记录中提取 txid，不展开 `[]SPOutput`。全部花费后的历史交易仍可定位，不要求 SDK 下载整个区块。

随后直接调用既有接口：

```text
POST /v3/bitcoin/rawtx/batch
{"txids": ["..."]}
```

SDK 检查每一项 error、实际 txid、真实 vout、完整公钥和金额；顶层 code=0 不代表所有条目成功。限额仍为单次 500 项。没有新增重复 rawtx 接口，也没有把 `getblock(...,3)` 作为客户端批量交易查询接口。

### 指定快照的输出状态

```text
POST /v1/sp/outpoints/query
{
  "snapshot_id": "...",
  "positions": [
    {"height": 900000, "block_hash": "...", "tx_index": 1,
     "vout": 2, "expected_txid": "..."}
  ]
}
```

返回 anchor 和逐项 state（unspent/spent/unknown）、txid、error。只针对已匹配候选；在同一固定 view 内验证创建位置、Archive、Base UTXO 与 Live 位图，不能用 bitcoind 当前 gettxout 或 mempool 代替。错误位置、错链、缺档和读取失败不是 spent。

### 请求保护

SP handler 沿用现有 Service 和 BaseResp 错误字段，保留 4 个并发 SP 请求的资源限额及 30 秒超时，由同一 Service 内的短 helper 实现，不增加独立 Gin 中间件或服务。POST 最大 256 KiB，最多 500 项，拒绝未知 JSON 字段和非法参数；响应完全编码后上限 16 MiB。状态码与既有 SP 协议保持一致：400 参数/响应超限、409 快照失效、429 资源限制、503 尚未完成初始同步、408 取消或超时。**BaseIndexer 处理区块、持久化或 CheckSelf 遇到致命错误仍直接 panic**。HTTP 请求中的读取 panic 交由原有 Gin Recovery 处理，当前请求失败（通常为 HTTP 500），进程继续运行；没有强制进程退出的 SP 中间件。

底层既有 Bitcoin RPC API 并非全部支持 context 取消；这里的请求上下文不等价于能强制取消正在进行的远端 Core RPC。部署仍需配置既有 RPC 连接超时。

## 5. CheckSelf

SilentPay 提供两个明确的自检入口：

| 入口 | 用途 | 检查范围 |
| --- | --- | --- |
| `BaseIndexer.CheckSilentPay(ctx)` | 日常跑数据，由既有 `BaseIndexer.CheckSelf()` 自动调用 | 区块目录完整性、候选输出身份及 Live/Base 未花费状态闭环 |
| `BaseIndexer.CheckSilentPayFull(ctx)` | 手工测试、完整本地数据库验证时显式调用 | 日常检查，加上每条 Archive 的 `T` 曲线合法性和未花费输出与 Base 地址脚本的公钥比对 |

两者所有读取均来自同一已提交 Base/SP 快照，**不访问 Bitcoin Core、其他网络服务或外部历史交易数据**；不新增配置、持久化编码、并发或后台服务。各自检查范围内的不一致、底层读取错误或取消仍立即 panic，不自动修复。SP 未初始化时直接调用任一 SP 自检入口均返回 `ErrSPNotInitialized`；日常 Base 自检继续跳过未初始化的 SP。

手工调用使用已初始化、已提交到目标高度的同一 BaseIndexer 实例：

```go
err := baseIndexer.CheckSilentPayFull(context.Background())
```

Archive 和 Live 在同一固定快照中，分别按键顺序以每批最多 1024 条、约 8 MiB 分页，再按 `(height, txIndex)` 顺序合并核验。每个游标复用页内字节缓冲和行偏移，不逐行分配 key/value；当前页处理完才复用该页内存。分批处理后再读取相关 Base 数据，避免在迭代器回调内嵌套读取造成死锁，且不把全部 Archive/UTXO 装入内存，不增加并发。核验 `sp/b/` 区块目录连续覆盖、与 Archive 交易位置精确一致、原始紧凑 SP 记录完整可解析、无多余或遗漏已登记的候选。检查起点、所有已登记区块的目录和已提交末块锚点；对没有 SP 初始化数据的旧 Base DB 则沿用前述禁用行为。

日常检查的核心不变量是：**Live 中标记未花费的输出，精确等于 Archive 已登记候选中仍存在于 Base UTXO 的输出。** 逐个候选输出从现存 `u-` Base UTXO 推导应有的 Live 位图，验证 UTXO 创建位置 ID，并与对应 `sp/l/` 行精确比对；不读取 `ai-` 地址资料，不执行 `T` 曲线点验证。仍保留记录长度、版本、压缩前缀、输出数量、vout 顺序/范围和完整编码边界检查。

完整检查额外对每条 Archive 执行压缩 `T` 公钥曲线合法性校验，对每个未花费输出读取 `ai-` 地址资料、还原 P2TR 脚本并逐字节比较输出公钥，保留原完整自检范围。查询和消费路径继续使用严格解析；日常自检的简化不影响这些路径。

两者 UTXO key 都直接从 Archive 的 txid 字节构造，保持原有字节序和十进制 vout 格式；只有报错时才构造 outpoint 文本。Live 不逐交易点查，也不单独进行末尾计数扫描；顺序合并直接拒绝缺行、位图不一致、完全花费交易残留 Live、位置提前或末尾剩余的孤立 Live，以及无效键。整个过程只读、不清理坏记录、不重新生成 `T`。

进度日志中的 `tx`、`live tx`、`outputs` 和 `elapsed` 均为本次连续自检的累计值，其中 `outputs` 是未花费候选输出数。约每经过 10000 个区块打印一次进度，不会因此从头重新扫描。进度及通过日志中的 `mode=core` / `mode=full` 标明实际检查范围。

**已确认的校验边界：** 日常检查通过仅表示目录及候选输出身份/存活状态闭环一致；曲线点或输出公钥内容损坏、地址资料损坏或缺失，但身份及位图仍对应时，可以通过日常检查，由完整检查覆盖。两种检查都不能独立重新计算 BIP352 `T`、验证原始区块 Merkle/Witness Commitment，亦无法找出目录和 Archive 双方同时遗漏的历史候选（尤其已完全花费的交易），因为 Base DB 未保存所有原始交易 Witness 和历史 Prevout。格式合法但数值错误的 `T` 不能仅凭 DB 判断；这部分由独立 BIP352 向量及真实 Bitcoin Core E2E 负责。“完整”指现有本地数据库可执行的全部检查，不代表重新验证原始区块。没有以总计数替代已登记记录与 Base UTXO 的逐项校验。

2026-10-25 用户提供的主网日志显示，拆分前的完整 DB 自检在 911200 高度检查 153057447 笔 Archive、45489248 笔 Live、57504393 个未花费候选输出，耗时 41 分 12.704 秒。新的日常入口可省去逐 Archive 曲线校验和逐未花费输出地址读取/解码；尚未在该生产快照实测拆分后的耗时，不能根据小型夹具推断生产耗时。

## 6. 新库与发布

本版只按已确认方案从空数据库顺序重建；不实现旧生产库 SP 在线迁移、独立回填或旧版本兼容状态。开发测试通过后再安排完整主网重跑，使用新目录，保留原服务/原数据库，不自动切换或删除旧库。

正常索引重用 Base 原始区块和已解析输入；钱包命中后的原始交易验证仍需可用的历史原始交易源，两种 CheckSelf 均不依赖该来源。bitcoind 必须具备索引与钱包验证所需历史区块和按 txid 查询能力，通常需要完整存储和已就绪的 txindex。剪枝或历史查询失败不能解释为“没有 SP 交易”。

发布前除默认全仓测试外，还应完成实际采用的 Badger 构建/测试、race 检查、独立 Bitcoin Core regtest 联调、完整主网重建 CheckSelf，以及 SDK/PWA 实际恢复验收。当前代码测试结果不替代这些部署验收。

## 7. 测试入口与本次执行边界

见 `e2e-testing.md`。默认 `go test ./...` 会运行 SP 单测和 E2E，无额外 SP tag，不访问公网，不读取生产配置或数据库，不广播真实交易。

2026-10-11 内存与批量查询优化：先用回归测试复现 4096 输出下 Clone 和稀疏消费的逐输出分配，再切换紧凑 pending 表示、来源分组读取及 RPC 副本释放；新增磁盘字节一致性、CompactSize 边界、原输入顺序、重复花费和多来源状态检查。随后对批量请求补红绿测试：优化前同一批 outpoints 查询 Archive 读取 7 次、txid 查询读取 200 次；优化后每个请求仅按不同来源读取 2 次（两笔创建交易）。`go test ./...` 已通过；独立 bitcoind、Badger-tag、race 和主网重跑不属于已执行项目。

官方参考：BIP352 规范及 bitcoin/bips 仓库 `bip-0352/send_and_receive_test_vectors.json`。目前测试内固化了已核对的官方双输入 T 基准，并测试顺序、异常输入和独立发送/接收签名闭环；没有将完整官方 JSON 向量集全部 vendoring 并宣称全量执行。
