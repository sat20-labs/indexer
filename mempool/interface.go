package mempool


/*
已经进入mempool的未确认TX的获取：在TX进入mempool时拿到这个TX的信息（钱包需要）。
参考btcwallet中的mempool代码github.com/btcsuite/btcwallet/chain，
在索引器中，维护一个mempool中所有未确认TX的列表。
1. 维护所有提交到mempool中的TX(包括输入和输出完整数据)
2. UTXO有四个主要状态，已经花费（不可花费），已经确认（可花费），花费但是未确认（输入），生成但是未确认（输出）。
   sat20索引器只提供了第二种状态，需要内存池提供后面两种状态。
3. 内存池中TX的数据需要本地保存数据库，提供快速查询接口（根据utxo查询，根据tx查询）
4. 内存池中TX在block生成时，需要从数据库中删除对应记录（根据输入的utxo删除相关记录）
*/
