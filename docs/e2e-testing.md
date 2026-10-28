# Indexer 隔离 E2E 测试

日期：2026-10-11。

## 目标与范围

在标准 `go test` 内验证从 Bitcoin 原始区块到 Base 持久化、HTTP 扫描、客户端匹配及可花费私钥的闭环。不建立第二个索引器，不添加独立测试运行框架或 SP 专用 build tag。

当前入口位于 `indexer/base` 的外部测试包 `base_test`：

- `silentpay_e2e_fixture_test.go`：确定性小链、交易生成/签名、临时数据库、真实 Base 同步辅助函数。
- `silentpay_e2e_test.go`：真实 SP HTTP 接口、已有 rawtx/batch 接口、现有核心场景与 Base 缓冲提交闭环。
- `silentpay_legacy_test.go`、`silentpay_legacy_e2e_test.go`：旧 Base DB 没有 SP 历史时正常继续同步、提交与 `CheckSelf`；五个 SP API 返回 503 `silentpay_not_initialized`，重启不生成局部 SP 索引；已有 SP 键、异常 schema 和未完成提交仍中止。

外部依赖仅替换 `bitcoin_rpc.ShareBitconRpc` 的数据源；其余使用生产代码。所有数据写入 `t.TempDir()`，HTTP 使用 `httptest.NewServer` 的本机临时端口，测试结束关闭服务、释放快照并删除临时目录。全局 RPC/网络变量测试后恢复，涉及这些全局变量的场景不得 `t.Parallel()`。

这是一套组件端到端测试，不是“启动了真实 Bitcoin Core”的集成测试。小链具有真实序列化交易、txid、区块头 PoW、txid Merkle root、witness commitment；实际付款和 SP 花费通过 Bitcoin 脚本引擎验证签名。但是完整共识、真实 Core RPC 服务与独立进程生命周期不由这个夹具模拟。后续 Core regtest 验收应作为另外明确标识的部署测试，不能把本结果冒充该项通过。

## 执行

在仓库根目录：

```sh
go build ./...
go test ./...
```

默认全仓测试会运行下列场景，不需要 `-tags e2e` 或环境开关。

只运行现有 SP E2E 用例：

```sh
go test ./indexer/base -run '^TestSilentPayE2E' -count=1
```

这只是标准 Go 测试筛选，不是不同的执行路径。

## 已实现的场景

| 用例 | 核心断言 |
| --- | --- |
| `TestSilentPayE2EReceiveLiveRecoverAndSpend` | 从头同步小链；真实 HTTP/CBOR 分页；错误 scan key 不匹配；同交易 k=0 花费后仍能恢复 k=1；旧快照不漂移；固定快照 outpoints 状态；按位置获得 txid；复用 rawtx/batch 及逐项错误；扫描派生私钥能签名花费；最后花费删除 Live 但保留 Archive |
| `TestSilentPayE2EReorgReloadRestoresLive` | 未提交分支消费后发生重组；真实 Base 检测分叉、重新 Init 并重放；原输出重新出现在 Live；旧快照不可复用；CheckSelf 通过 |
| `TestSilentPayE2ECheckSelfDetectsCorruptionWithoutRepair` | 同时缺失 Archive/Live、错误 Live 位图及畸形 T 编码仍使 CheckSelf panic；Bitcoin RPC 不可用时纯 DB 自检仍正常进行；检查不得修改数据 |
| `TestSilentPayE2EDatabaseReopen` | 真实关闭数据库并重新打开；新 Base 恢复持久化 Live；旧 token 失效；重新扫描获得私钥并验证花费签名 |
| `TestSilentPayE2ESameBlockSpendAndFullySpentHistoryAudit` | 创建与全部 SP 花费在同一块；Live 为空而 Archive 保留；已确认纯 DB 自检无法发现 Archive 与目录同时遗漏的已完全花费候选（不再伪称独立读取原始区块） |
| `TestSilentPayE2EBaseBufferedCloneSubtractCommit` | Base Clone(true) 冻结、处理后续花费、Subtract、先提交冻结批次、SetSyncStats、提交后续增量，并重复空提交；核验 Archive、Live 与 Base.CheckSelf |

生产扫描组件不会取得测试钱包的私钥。发送端使用独立公式生成输出，接收测试用公共 T 和本地 scan/spend 私钥匹配；最终签名由生产 Bitcoin 脚本引擎验证，不能只断言“算出了相同字符串”。

HTTP 测试改为直接使用 `rpcserver/base.NewService(...).InitRouter()` 和已有 `rpcserver/bitcoind.Service` 的实际路由，不再通过独立 `NewSilentPayService` 挂载。测试使用仅嵌入接口的 SP 能力适配器，避免为 SP 测试模拟无关 Base 方法。生产挂载仍由现有 Base router 完成，实际服务通过 IndexerMgr 读准入锁调用 Base；夹具不启动 Runes/BRC20 等不相关的全量模块。

`/v1/sp/status` 的 E2E 额外断言状态查询不调用 bitcoind.GetRawTx；真实原始交易获取和逐项失败仍通过 `rawtx/batch` 验证。

## 紧凑记录与内存回归

`indexer/base/silentpay_memory_test.go` 复用已有临时数据库和 SP 状态夹具，避免引入新的性能测试框架：

| 用例 | 核心断言 |
| --- | --- |
| `TestSilentPayCloneDoesNotAllocatePerOutput` | 4096 个输出的不可变 pending 记录不逐输出深拷贝；Clone 的对象分配上限为 128 |
| `TestSilentPaySparseSpendDoesNotExpandAllSourceOutputs` | 只消费同一来源的首尾两个输出，Archive 只读取一次；不展开全部历史输出对象；分配上限为 256 |
| `TestSilentPayRPCCloneDoesNotRetainPendingRecords` | RPC 副本保留共享数据库/视图池，但不持有 pending Archive/Live，也不改变编译实例 |
| `TestSilentPayCompactRecordsSurviveCommitAndRelease` | v1 磁盘字节与既有编码完全一致；Clone/Subtract/落盘后 pending 引用释放，后续扫描仍完整保留已花费输出和正确位图 |

先在优化前运行了这些测试，实际复现 4096 输出下 Clone 分配 4104 次、稀疏消费分配 4119 次及 RPC 副本保留 pending 数据；优化后的默认全仓测试验证这些回归通过。上限断言用于防止重新引入逐输出对象分配，不代表主网峰值内存或吞吐实测。

`indexer/base/silentpay_records_test.go` 进一步覆盖 CompactSize 差分在 252/253/65537 处的编码、截断/尾随损坏、多个大来源交错消费只读一次且保持 Base 原输入顺序、重复消费必须报错。相关用例与其他测试一样默认执行，不加特殊 tag。

内存优化保持磁盘 schema 和 HTTP 字段不变。数据库 value 仍整行读取；按需处理指只读取实际使用的来源，并流式遍历其紧凑数据，不建立全量输出对象缓存。测试不将其描述为底层支持字节范围读取。

`indexer/base/silentpay_batch_query_test.go` 使用现有临时 Base 数据库和实际 Pebble/Badger 只读视图，附加只读计数包装器，覆盖：同一来源大量输出的稀疏 outpoint 查询（混合 spent/unspent、重复请求、错误链 hash、错误 txid、非法范围和缺失 vout），多来源混排仍按原输入顺序返回，以及 txid 批量定位不展开不必要的输出。额外破坏紧凑记录尾部，确保即使命中前部 vout 也会因整条记录校验失败。红灯时同批 outpoints 查询读 Archive 7 次、200 项 txid 查询读 200 次；修复后各仅按两笔来源读取 2 次，Live 每个来源最多一次。没有新的 E2E 架构或特殊 build tag。

## SP 数据库只读 CheckSelf（2026-10-20）

遵照已确认的约束，两个入口在 SP 部分仅检查**现有 Base/SP 数据库**，不调用 Bitcoin Core 网络 RPC，不读取区块或历史父交易。2026-10-25 起，默认 `BaseIndexer.CheckSelf()` 调用日常入口 `CheckSilentPay(ctx)`，用 Base UTXO 的存在性和创建位置 ID 独立重建 Live 位图，并校验目录/Archive 完整对应；手工调用 `CheckSilentPayFull(ctx)` 额外执行 `T` 曲线验证和 Base 地址脚本公钥比对。两者共用固定已提交快照上的顺序分页和闭环检查，批次上限为 1024 条或约 8 MiB，避免在数据库迭代器持锁时嵌套读取。

新增默认测试：

- `silentpay_checkself_perf_test.go`：模拟数据源封锁全部 Bitcoin RPC，验证日常 CheckSelf 和完整 SP 检查均不请求区块、交易或父交易。
- `silentpay_checkself_paging_test.go`：2500 条 Archive/Live 记录交错跨页、顺序/无重复、固定快照之后写入不影响两个游标、中途取消、大记录触发字节上限分页、底层扫描错误不能发布半页数据。
- `silentpay_checkself_merge_test.go`：2500 笔交易混合完全花费、部分花费与未花费状态，Archive/Live 在不同位置跨页，两个入口的 Live 点查次数都必须为零；日常/完整入口都读取 5000 个候选 UTXO，地址读取分别为 0/2500；缺失、孤立、空或错误位图、截断/尾随字节、缺失 Archive/目录在两种检查中仍失败且不修复数据，非法曲线点由完整检查拒绝；未初始化、取消和未完成提交保护；空 Archive 的孤立 Live 检测；直接构造的 UTXO key 与既有编码逐字节一致。`BenchmarkSilentPayDBAuditPaging` 在真实后端上单独测量 4096 条记录的分页耗时和分配，不代表完整主网自检耗时。
- `silentpay_db_corrupt_e2e_test.go`：通过真实 Base.CheckSelf 和手工完整入口验证：孤立 Live/Archive、缺少 SP block、缺失 Base UTXO、错误 UTXO ID 和结构损坏 Archive 在两种检查中仍失败；曲线点非法、输出公钥不匹配、地址资料损坏/缺失在日常检查中按已确认范围放行，完整检查仍拒绝；公共查询继续拒绝非法曲线点；测试后数据库恢复可重新通过。

两个入口均**不**重新计算 `T`，不验证原始区块的 Merkle/Witness Commitment，也不能发现 SP block 目录和 Archive 双方同时漏掉的已花费历史候选。日常检查通过表示目录及身份/存活状态闭环一致；完整入口保留之前全部本地 DB 检查。两者都不能把缺失 UTXO 解释为成功而忽略与 Live 的矛盾。

默认 `go test ./...` 和 `go build ./...` 已通过；尚未使用用户的 testnet4 155027 高度原始数据库测量耗时，也没有运行 Badger-tag 和 race 专项。不能仅凭小型测试夹具给出分钟级性能承诺。

2026-10-24 顺序合并优化验证：实际部署后端为 Badger。`go test -tags badger ./indexer/base ./indexer/db ./rpcserver/base ./common -count=1` 已通过，包含既有 SilentPay E2E、零 Bitcoin RPC 检查和上述新增回归用例。2500 笔交易夹具的 Live 点查由原实现的 2500 次降为零；完整 Base UTXO/地址、Archive 曲线点与结构、目录及位图校验继续执行。未在生产数据库上测量完整自检耗时，本轮未运行 race 或独立 bitcoind 联调。

同日 Badger 分页微基准：`go test -tags badger ./indexer/base -run '^$' -bench '^BenchmarkSilentPayDBAuditPaging$' -benchtime=1s -count=3`，每轮扫描 4096 条、每条 value 为 100 字节。修改前通过 Go 临时源码 overlay 使用原实现运行相同 benchmark，工作区不回退。三次结果中，分配次数从约 14661 降至 6457（约减少 56%），分配字节从约 1.03 MB 降至 0.407 MB（约减少 61%）；耗时从 2.82–2.96 ms 变为 2.40–2.86 ms，样本不足以宣称稳定的耗时改善。此基准只测分页，不包含取消 Live 点查、UTXO key 构造或完整自检的收益。

2026-10-25 双入口拆分验证：先用读取计数测试复现日常入口仍读取 2500 次地址，修改后日常地址读取为零、完整入口仍为 2500 次。`go test -tags badger ./indexer/base ./indexer/db ./rpcserver/base ./common -count=1` 已通过，包括内容检查边界、两种检查的状态闭环损坏检测及零 Bitcoin RPC 回归。日志增加 `mode=core/full`，没有增加配置项、存储格式或并发。本轮未运行 Pebble、race、生产快照计时或独立 bitcoind 联调。

## 更低层的故障测试

`common/silentpay_test.go` 包含已核对的官方双输入 T 基准、输入顺序、非模板 P2PKH、重复公钥、聚合中间无穷点/最终无穷点、各支持输入类型、Taproot 奇偶性、NUMS 排除和未来 witness 版本等测试。当前不是完整官方向量 JSON 的全量测试运行器。

`indexer/base/silentpay_state_test.go` 验证 Clone/Subtract、Live 位图、历史 Archive、快照过期与非法游标、提交保护及畸形编码；已提交状态下 SP schema 丢失必须 panic。增加激活前区间不允许虚假完成锚点、候选目录存在但 Archive 缺行必须报损坏、目录明确非候选可跳过、同块多次消费同一创建交易只读取一次。

`indexer/base/silentpay_commit_test.go` 提供测试专用 KVDB 包装器，故意让 WriteBatch 每一条操作独立落盘，再注入失败，覆盖：

- 提交前标记 Sync 失败。
- 部分子批次成功后下一条 Put 失败。
- Batch Flush 失败。
- 数据落盘阶段 Sync 失败。
- 删除提交保护标记失败。

每个场景都验证：未完成标记保留，新的 read view 被拒绝，重新初始化不信任半完成库；故障前已固定的完整只读视图仍是原状态。修复原入口测试因 Base 未初始化引起的空指针假通过：`TestSilentPayBaseUpdateDBPanicsAfterPersistenceFailure` 通过真实 Base.Init、注入点命中标记和 panic 内具体注入错误验证五种失败阶段均走到预期位置；成功对照验证保护标记仅在全部持久化成功后移除。

这里的失败由返回错误注入，没有实际拔电，也不宣称替代真实 Badger 下的崩溃/断电测试。

## 复用方法

新增 Base 派生索引场景时，先复用现有小链和数据库辅助函数，不另起监听端口规则、磁盘数据目录、后台守护进程或依赖生产配置。需要新增输入脚本时，用真实签名和脚本验证构造；需要修改区块时同时维护 txid Merkle/witness commitment，避免把无效 fixture 导致的失败误判为索引错误。

新用例应明确观察公开入口和持久化结果，并验证失败路径。不能直接向数据库注入期望成功结果，再把读取成功算作端到端通过。故障注入改数据库仅用于验证默认 CheckSelf 能检测损坏且不自动修复。

## 本次验证与剩余发布验收

`rpcserver/base/silentpay_fatal_test.go` 改用标准 `NewService` 路由，验证 HTTP 使用 Gin 原有 Recovery：请求处理中的 panic 返回 500，但进程保持运行并可以处理后续请求。BaseIndexer 的持久化失败仍 panic，不在 HTTP 层增加强制退出中间件。所有 E2E 随默认 `go test ./...` 运行，不使用特殊 build tag。

按确认范围，P2PK/裸多签的原始脚本还原缺口仍然存在，不修改 Base 地址表示；纯 DB CheckSelf 不再调用 Bitcoin Core 或独立重算 BIP352 `T`，因此不再覆盖原始交易 Witness/Merkle 验证或双方同时遗漏的候选。它仍严格检查库内一致性，所有独立密码学正确性由现有发送/接收向量与 E2E 负责。不增加 SyncStats 意外丢失的异常处理。测试通过不表示已排除的场景获得支持。

下列命令是后续验证入口，本次尚未执行，不能写入“已通过”清单：

```sh
go test -race ./common ./indexer/base ./rpcserver/base
go test -tags badger ./...
```

`badger` 是仓库既有数据库后端选择，不是新 E2E 用例的特殊 tag；相同 E2E 源码没有 SP build tag。当前默认后端测试不能证明实际 Badger 构建已经通过。

尚需独立 Bitcoin Core regtest、完整官方向量集合、真实主网从头重建与完整 CheckSelf、主网磁盘/IO/快照保留性能，以及 SDK/PWA 实际联调。本次没有读取生产钱包、启动主网重跑、广播真实交易、切换服务或删除旧数据库。
