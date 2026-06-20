# ARC-20 与 Atomicals 主协议技术规范

> 说明：本文按当前 `indexer/atom` 代码行为整理，不把它写成“理想中的协议说明”。凡是和官方源码或本仓库实现一致的内容，才作为规范描述。

## 1. 协议定位

Atomicals 可以理解为一套把“资产状态”锚定在 Bitcoin UTXO 上的协议族。它的核心不是独立账户，而是：

1. 通过比特币交易表达协议动作。
2. 在 witness 里携带 Atomicals payload。
3. 用 UTXO、holder、ticker、mint history 共同描述资产状态。
4. 通过激活高度分阶段引入新规则。

在本仓库里，ARC-20 相关资产主要对应可替代资产（FT）和其铸造、拆分、颜色标记等行为。

## 2. 交易载荷与解析方式

当前解析器从交易输入 witness 中寻找 Atomicals 结构：

1. witness 里出现 `OP_IF`.
2. 紧跟 magic 字节 `atom`.
3. 再读取操作码。
4. 之后解析 CBOR payload。

解析器当前识别的 opcode 包括：

- `nft`
- `dft`
- `dmt`
- `ft`
- `sl`
- `y`
- `z`
- `x`
- `mod`
- `evt`
- `dat`

其中和 ARC-20 最直接相关的是：

- `ft`：直接 FT 铸造
- `dft`：部署 DFT
- `dmt`：执行 DFT mint
- `y`：split
- `z`：custom color

## 3. 激活高度

当前主网激活规则在代码里是硬编码的。下面是本仓库识别到的主网关键高度：

| 高度 | 能力 |
|---|---|
| `808080` | 主协议激活 |
| `819181` | `dmint` 激活 |
| `822800` | `commitz` 激活 |
| `828128` | `density` 激活 |
| `828628` | `rollover` 激活 |
| `848484` | `coloring` 激活 |

### 3.1 `808080`

这是主协议基线高度。低于该高度，不进入 Atomicals 转移与铸造逻辑。

### 3.2 `819181` - dmint

这是 DFT 相关 mint 规则的一个分水岭。代码里可以看到它影响到：

- payload 是否允许更复杂的字段
- mint 相关参数的约束
- 后续 DFT 的可配置范围

### 3.3 `822800` - commitz

这是承诺与提交相关能力的分界点。实际实现里，ticker 注册、mint 约束和提交参数会依赖这类高度条件。

### 3.4 `828128` - density

这是一个非常关键的升级点。它影响：

- DFT 的 `max_mints` 约束上限
- 是否允许更高密度的 mint 结构
- 是否允许更复杂的 bitwork / mint 参数组合
- payload 校验路径

### 3.5 `828628` - rollover

这类高度主要影响 bitwork 和 mint 的延续规则。简单说，它决定某些“延续型”mint 规则是否可以成立。

### 3.6 `848484` - coloring

这是颜色分配规则的关键分界点。代码里，`customActivated := block.Height >= s.heights.CustomColoring`，说明：

- 低于该高度，分配逻辑偏“常规”
- 高于或等于该高度，custom coloring 逻辑生效

这会直接影响 split / color 相关输出归属。

## 4. ARC-20 的状态模型

ARC-20 在这个实现里不是单一表，而是几个状态面叠加：

### 4.1 Ticker

Ticker 记录一个资产的定义信息，包括：

- ticker 名称
- atomical id
- deploy height
- mint height
- mint amount
- max mints
- minted amount
- minted times
- mint mode
- bitwork 约束

### 4.2 UTXO

UTXO 记录每个输出上承载了多少资产。它是最终可验证的账本视图。

### 4.3 Holder

Holder 是地址聚合后的视图。它不是协议原语，而是索引器对资产分布的汇总。

### 4.4 Mint history

Mint history 记录每次 mint 的历史事件，方便追踪来源和回放。

## 5. 地址归一化

当前仓库对 holder 地址的处理原则比较明确：

1. 先查地址数据库。
2. 如果输入看起来像 hex script，则尝试 `ExtractPkScriptAddrs`。
3. 如果是 segwit 地址，则先恢复 script，再做同样处理。
4. 最后才回退到 script 的 base64 表示。

这意味着：

- 地址不是协议本体的一部分，而是索引器的解释层。
- 做校验时，应当固定一套地址归一化规则。
- 不建议为了局部 checkpoint 去改全局 `GetAddress()` 语义。

## 6. 约束规则

当前实现里可见的约束包括：

- ticker 名称长度和字符集受限
- DFT mint amount 有上下限
- `max_mints` 在不同激活高度下有不同上限
- mint height、bitwork、payload 字段类型都会参与校验

这些约束不是装饰性规则，而是决定一个 mint/部署是否有效的硬条件。

## 7. 本仓库对 ARC-20 的索引结论

就当前代码来说，ARC-20 的索引关键点有三条：

1. **交易输入 witness 是协议入口。**
2. **激活高度决定规则分支。**
3. **UTXO 和 holder 只是同一资产状态的两个投影。**

因此，做 ARC-20 的索引器，核心不是“识别某个 ticker 名字”，而是正确处理：

- 协议 payload
- 高度分支
- UTXO 转移
- holder 汇总
- checkpoint 对账

