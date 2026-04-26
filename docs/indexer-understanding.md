# Workspace Understanding

> 记录时间：2026-04-26。本文是 Codex 对当前 workspace 的项目理解，供后续协作时持续校准和补充。
>
> 跨项目关系记录见 `satoshinet/docs/core-projects-understanding.md`。当前核心关注范围是 `indexer`、`docs`、`docs-en`、`sat20wallet`、`transcend`、`satoshinet`。

## 项目定位

这是 `github.com/sat20-labs/indexer`，一个 Go 实现的 sat20 / ordx 协议索引器。核心能力是从 bitcoind 拉取 Bitcoin 区块，维护基础 UTXO、地址、sat ordinal range 状态，并在此基础上索引 Ordinals 铭文、ORDX FT/NS/NFT、BRC-20、Runes、稀有聪等资产数据，然后通过 Gin HTTP API 提供查询和交易辅助接口。

仓库是单 Go module，主要依赖 btcd/btcwallet 的 sat20-labs fork 以支持 testnet4；数据库默认用 Pebble。README 很短，安装和运行信息主要在 `install/guide.md`、`example.env.yaml`、`install/conf_*.yaml`。

## 入口与运行方式

生产入口是仓库根目录的 `main.go`，不是 `cmd/main.go`。根入口流程是：

1. `config.InitConfig("")` 读取 `.env` 或 `-env` 指定 YAML。
2. `config.InitLog` 初始化日志。
3. `InitRpc` 初始化共享 bitcoind RPC：`share/bitcoin_rpc.ShareBitconRpc`。
4. `indexer.NewIndexerMgr(yamlcfg)` 创建全局单例 manager。
5. `base_indexer.InitBaseIndexer(indexerMgr)` 暴露共享基础索引器接口。
6. `indexerMgr.Init()` 打开 DB、初始化所有子索引器。
7. `InitRpcService` 在 `max_index_height <= 0` 时启动 HTTP API。
8. `indexerMgr.StartDaemon(stopChan)` 开始同步循环。

`cmd/` 目录更像临时工具/对比测试入口；当前 `cmd/main.go` 直接调用 `TestCompareHolders()`，不能当作服务入口理解。

## 配置与外部依赖

核心配置类型在 `config/config.go`：

- `chain`: `mainnet`、`testnet`、`testnet4`。
- `db.path`: 各子 DB 的根目录。
- `share_rpc.bitcoin`: bitcoind RPC host/port/user/password。
- `basic_index.max_index_height`: 大于 0 时跑到指定高度、检查并退出，通常用于阶段性编译和备份 DB。
- `basic_index.period_flush_to_db`: 编译期间周期性 flush。
- `rpc_service.addr/proxy/log_path/swagger/api`: HTTP 服务配置。
- `pubkey`、`check_validate_files`: 供部分业务校验使用。

bitcoind RPC 封装在 `share/bitcoin_rpc`，全局变量 `ShareBitconRpc` 被 `indexer/base`、`runes`、`rpcserver/bitcoind`、`mempool` 等多处调用。

## 核心模块地图

### `common/`

跨模块共享的数据结构和工具层。这里定义了：

- 区块、交易、UTXO、sat range、资产名、资产数量、ticker、NFT/NS/BRC20 类型。
- `KVDB`、`ReadBatch`、`WriteBatch` 抽象。
- ordinals/satsnet 资产解析、decimal、日志、常量和错误。
- protobuf 生成类型在 `common/pb`。

后续改动类型字段时，影响面会很大，因为 indexer、rpcserver、db 序列化都会引用这里。

### `indexer/base/`

基础链索引器。主要职责：

- 从 bitcoind 拉块：`FetchBlock`。
- 把 btcutil block 转为内部 `common.Block`。
- `SyncToChainTip` / `syncToBlock` 驱动连续同步。
- `prefetchIndexesFromDB` 预加载输入 UTXO 和地址状态，降低随机读成本。
- `assignOrdinals_sat20` 给交易输入输出分配 sat range。
- 维护基础 DB：block、utxo、address、addressId、utxoId。
- 通过 `BlockProcCallback` 回调上层协议索引。
- 检测 reorg，保留 mainnet 6 块、testnet4 72 块历史窗口。

`RpcIndexer` 是 BaseIndexer 的服务快照，用于 API 查询时隔离正在编译的内存状态，并通过 `deletedUtxoMap` / `addedUtxoMap` 补齐尚未落库的增量。

### `indexer/indexermgr.go`

`IndexerMgr` 是总协调器和对外查询接口实现者。它持有所有 DB 和子索引器：

- `base`: 基础 UTXO/address/sat range。
- `exotic`: 稀有聪资产。
- `nft`: Ordinals 铭文/NFT。
- `ns`: 名字服务。
- `ftIndexer`: ORDX FT。
- `brc20Indexer`: BRC-20。
- `RunesIndexer`: Runes。
- `miniMempool`: 未确认交易小缓存。
- `rpcService`: 面向 API 的 BaseIndexer 快照。

它的重要策略是延迟落库抗 reorg：当前高度和 DB 同步高度之间保持一个 history gap。`prepareDBBuffer` 克隆各子索引器，`performUpdateDBInBuffer` 写入备份实例，`cleanDBBuffer` 从实时实例里剪掉已写入部分。这样 DB 最多落后当前高度若干块，遇到 reorg 可以关闭 DB、重新 Init、从安全高度继续。

### `indexer/handle.go`

这是协议处理主链路。`BaseIndexer` 每处理一个块后回调 `processOrdProtocol`：

1. `exotic.UpdateTransfer` 先生成稀有聪资产，为 ORDX 依赖准备数据。
2. 高度未到 `ordFirstHeight` 时直接返回。
3. 遍历每个 tx input witness，用 `ord0_14_1.GetInscriptionsInTxInput` 提取铭文。
4. `handleOrd` 解析铭文和协议内容，写入 NFT/NS/FT/BRC20 等模块的区块内缓存。
5. 按顺序执行转移更新：
   - `nft.UpdateTransfer`
   - `ns.UpdateTransfer`
   - `brc20Indexer.UpdateTransfer`
   - `RunesIndexer.UpdateTransfer`
   - `ftIndexer.UpdateTransfer`

这个顺序很关键：FT 依赖前面生成的稀有聪和 NFT 信息，BRC-20 也依赖 NFT 铭文解析结果。

### 子协议索引器

- `indexer/nft`: Ordinals 铭文/NFT 索引。维护 sat -> NFT、utxo -> sat、content/contentType、inscriptionId -> nftId 等映射。它是 NS、FT、BRC20 的基础依赖。
- `indexer/ns`: 名字服务，依赖 `nft.NftIndexer`。
- `indexer/ft`: ORDX FT，依赖 NFT 和 exotic 稀有聪。维护 ticker、holder、utxo asset map、mint history。
- `indexer/brc20`: BRC-20，依赖 NFT。维护 ticker、address holder、transfer NFT、action history，并带 CSV checkpoint/validate 支持。
- `indexer/runes`: Runes，依赖 BaseIndexer 和 runestone parser。表结构分布在 `indexer/runes/table`，写入经 `store.DbWrite` 缓存和日志批量提交。
- `indexer/exotic`: 稀有聪/特殊 satribute。它在区块处理早期运行，后续 ORDX FT 会用到。

所有这些模块基本都实现了 `Init`、`Clone`、`Subtract`、`UpdateTransfer`、`UpdateDB`、`CheckSelf` 这一组生命周期。

## HTTP API 层

`rpcserver/router.go` 创建 Gin engine，配置 CORS、公共安全 header、压缩、swagger，然后挂载四组服务：

- `rpcserver/base`: 健康检查、稀有聪类型、plain/all UTXO。
- `rpcserver/ordx`: 主要业务 API，包含地址资产摘要、UTXO 资产详情、ticker、holder、mint history、NS、NFT、v3 资产接口、KV 注册/读写。
- `rpcserver/ord`: ord 内容预览和静态资源。
- `rpcserver/bitcoind`: send/test tx、raw block/tx、fee estimate 等 bitcoind 代理接口。

接口依赖的是 `share/base_indexer.Indexer`，`IndexerMgr` 实现这个接口。改 API handler 时通常应先找 `rpcserver/ordx/router.go` 路由，再进 `handler*.go`，最后落到 `indexer/*_interface.go`。

## 数据库层

DB 抽象在 `common/db_interface.go`，实现集中在 `indexer/db`。当前 `NewKVDB` 默认返回 Pebble。

`IndexerMgr.initDB` 会在配置根目录下打开多个独立 DB：

- `base`
- `nft`
- `ns`
- `exotic`
- `ft`
- `brc20`
- `runes`
- `local`
- `dkvs`

Pebble 参数偏向大索引编译：32GB cache、较大 memtable、Bloom filter、单线程 compaction。注释里明确区分编译期和服务期参数，但当前打开默认走 `buildOptions()`。

## Mempool / MPN / DKVS

`indexer/mempool.go` 是轻量 mempool 跟踪，用 bitcoind mempool 数据维护未确认花费、锁定 UTXO 等状态，供 v3 API 构造交易时过滤。

`indexer/mpn` 是更完整的 P2P/mempool node，很多代码来自 btcd 的 peer/connmgr/netsync/mempool/addrmgr 体系。但按当前项目状态，它没有实际接入运行主流程：`IndexerMgr.StartDaemon` 里启动 MPN 的代码被注释掉，现阶段主流程只应按 `MiniMemPool` 理解。

`dkvs/` 是基于 libp2p Kademlia DHT 的分布式 KV 能力。但按当前项目状态，这个目录中的代码也没有实际作为 indexer 主流程依赖使用。`indexer/interface_kv.go` 和 `rpcserver/ordx/handler_kv.go` 暴露了 KV 注册、put/get/del 一类 API 形状，但不要把 `indexer/dkvs` 当作当前运行时问题排查入口，除非任务明确要求研究这块。

## 关键数据流

```text
bitcoind RPC
  -> share/bitcoin_rpc.ShareBitconRpc
  -> indexer/base.FetchBlock
  -> BaseIndexer.syncToBlock
  -> prefetch UTXO/address from base DB
  -> assign sat ranges
  -> IndexerMgr.processOrdProtocol
  -> exotic / nft / ns / brc20 / runes / ft UpdateTransfer
  -> delayed Clone buffer
  -> UpdateDB into separate Pebble DBs
  -> IndexerMgr.updateServiceInstance clones RpcIndexer
  -> rpcserver handlers read through share/base_indexer.Indexer
```

## 高风险注意点

- `IndexerMgr` 是单例，测试或工具代码多次调用 `NewIndexerMgr` 会拿到同一个实例。
- `cmd/main.go` 不是服务入口，改启动流程要看根 `main.go`。
- `processOrdProtocol` 的模块顺序不要随意调整。
- `indexer/mpn` 和 `indexer/dkvs` 当前不是实际使用的主流程代码，不要把它们误判成运行依赖。
- DB 落库有延迟和 clone/subtract 机制，修状态写入时要同时考虑实时实例、备份实例、Subtract 后的残留。
- `rpcEnter` / `rpcLeft` 和 `reloading` / `rpcProcessing` 用来避免 reorg 重载与 RPC 查询并发冲突。
- 多个模块用 protobuf/gob/msgpack 混合序列化，改 DB value 类型时要找到对应 `db.go` / `dbkey.go` / `update.go`。
- 现有工作区有未跟踪本地产物：`4l-btc.txt`、`5d.txt`、`download.sh`、`indexer-mac`、`indexer-mainnet`、`indexer-testnet`、`nohup_testnet.log`。后续不要误删或纳入无关提交。

## 后续深入建议

为了更深入理解，下一步可以围绕具体任务继续追：

- 某个 API：从 `rpcserver/*/router.go` 到 handler，再到 `IndexerMgr` interface。
- 某个协议资产：从 `handleOrd` 的协议解析，到对应子索引器 `UpdateTransfer` / `UpdateDB`。
- 某个 DB 问题：先找该模块 `dbkey.go` 和 `db.go`，再看 `UpdateDB` 和 `Clone/Subtract`。
- 性能问题：优先看 `BaseIndexer.prefetchIndexesFromDB`、Pebble 参数、prefix scan、各模块 `UpdateDB`。
