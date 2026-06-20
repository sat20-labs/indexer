# 官方扩展功能与索引器支持方案

本文整理 Atomicals 官方源码里能看到的扩展能力，以及如果后续要让本仓库索引器支持这些能力，应该怎么设计。

## 1. 官方能力分层

从当前代码和 checkpoint 数据看，官方能力大致可以分成四层：

1. **基础资产层**  
   FT / DFT / mint / transfer / split / color。

2. **命名与对象层**  
   `nft`、realm、subrealm、以及其他对象式资产。

3. **扩展规则层**  
   dmint、commitz、density、rollover、coloring 这类高度驱动的规则。

4. **对账层**  
   官方 snapshot、ticker/holder/utxo 对比、checkpoint 目标高度校验。

## 2. NFT

### 2.1 代码现状

解析器已经识别 `nft` opcode，但本仓库当前的主索引逻辑重点仍然是 FT / ticker / UTXO / holder。

### 2.2 NFT 的索引含义

NFT 的关键不是余额，而是：

- 唯一对象标识
- 元数据
- 所属 UTXO
- 转移历史
- 可能的可见名称或展示字段

### 2.3 如果我们要支持 NFT

建议单独建立 NFT 状态表，而不是塞进 ticker 体系：

- NFT 对象表：object id、inscription/atomical id、当前 UTXO、当前 owner、metadata hash
- NFT 事件表：mint、transfer、burn、update
- NFT 验证层：按高度回放，确认对象唯一性和转移链

不要把 NFT 强行映射成 FT holder，否则会丢失对象语义。

## 3. Realm / Subrealm

### 3.1 代码现状

当前 checkpoint 数据里已经出现 `realm`、`realms`、`subrealm` 等命名相关 ticker。说明官方协议里，命名空间能力是实际存在的，而不是旁支概念。

### 3.2 Realm 的本质

Realm 不是单纯的 token，它更像命名空间对象：

- 名称注册
- 名称归属
- 子命名空间派生
- 层级关系
- 冲突和唯一性

### 3.3 如果我们要支持 Realm

建议增加三类结构：

- Realm 注册表：name、owner、root object id、deploy height
- Subrealm 归属表：parent realm、sub name、derived object id
- 规范化解析器：把 witness payload 解析成层级路径

Realm 不应该只靠 ticker 词典管理。它需要对象图和层级规则。

## 4. Mint 技术

### 4.1 Direct FT

`ft` 是直接 FT 铸造，典型语义是：

- 直接指定请求 ticker
- 一次性或固定数量铸造
- 资产直接进入输出

### 4.2 DFT deploy / mint

`dft` 负责定义规则，`dmt` 负责执行 mint。

DFT 的重点在于：

- mint_height
- mint_amount
- max_mints
- bitwork 约束
- 高度分支约束

### 4.3 Split 与 custom coloring

`y`、`z` 主要影响输出如何被分配。特别是 `848484` 之后，custom coloring 的行为要按新规则执行。

### 4.4 如果我们要支持更多 mint 规则

建议把 mint 规则拆成三层：

1. **语法层**：payload 能不能解析。
2. **规则层**：激活高度和参数是否满足。
3. **结算层**：最终如何写入 UTXO / holder / history。

这样后续加新 mint 形态时，不会把主索引器搞成一堆 if-else。

## 5. Checkpoint 与 validate 数据

### 5.1 当前做法

当前仓库已经把 checkpoint 和 validate 数据分开：

- `checkpoint.go` 负责最基础的对账入口和校验
- `validate/` 保存可对比的 CSV 数据
- `snapshot.go` 负责导出比较快照

这种结构是对的，因为它把“业务索引”和“校验材料”分开了。

### 5.2 关键原则

后续支持官方新功能时，建议继续坚持：

- validate 数据独立保存
- checkpoint 只保留最小入口信息
- 不要覆盖历史已验证数据
- 新规则必须能在指定高度重复回放

## 6. 给索引器的扩展建议

如果以后要完整支持官方 Atomicals 能力，推荐按下面顺序做：

1. **先稳定基础 FT 层**  
   保证 ticker、UTXO、holder、mint history 正确。

2. **再补对象层**  
   NFT、realm、subrealm 单独建模。

3. **再补高度驱动规则**  
   将 `808080`、`819181`、`822800`、`828128`、`828628`、`848484` 放进统一的激活配置。

4. **最后补对账基础设施**  
   为每个关键高度保存 ticker、holders、utxos 的 validate 数据，并支持回放比对。

## 7. 推荐的数据组织方式

建议继续保持如下目录思路：

- `indexer/atom/validate/tickers/`
- `indexer/atom/validate/holders/`
- `indexer/atom/validate/utxos/`
- 未来可扩展：
  - `indexer/atom/validate/nft/`
  - `indexer/atom/validate/realm/`
  - `indexer/atom/validate/events/`

文件命名建议继续按高度区分，便于回放：

- `atom-holders-860000.csv`
- `quark-holders-900000.csv`
- `atom-utxos-950000.csv`
- `tickers-950000.csv`

## 8. 结论

当前这套索引器已经具备了 ARC-20 的基础骨架。继续往官方能力靠齐，关键不是“把所有东西揉成一个大表”，而是：

- 解析层分清 opcode
- 规则层分清高度
- 状态层分清 FT / NFT / realm
- 校验层分清 validate 与 checkpoint

只要这四层分开，后续补功能的风险会低很多。

