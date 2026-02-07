索引器安装说明
====


尽可能自己拉取代码，并且本地编译。
这里提供的版本，仅用于 Ubuntu 22.04.4 LTS

在自己的数据区创建一个目录，比如 /data/indexer
将这indexer_1.0.2和两个conf.yaml文件拷贝到这个目录下
如果想同时跑主网数据和测试网数据，可以将indexer这个执行文件再复制一份，改名为
indexer_mainnet
indexer_testnet

修改conf.yaml，主要修改btc的用户名和密码.

最后，执行下面命令，开始跑主网数据。
nohup ./indexer_mainnet -env ./conf_mainnet.yaml > ./nohup_mainnet.log 2>&1 &

测试网：
nohup ./indexer_testnet -env ./conf_testnet.yaml > ./nohup_testnet.log 2>&1 &


测试网数据可能几个小时就完成，然后自动进入服务状态。可以打开浏览器，输入：http://127.0.0.1:8009/testnet/bestheight
查看索引器同步的最新高度。

主网数据一般要一周。为了确保数据不会因为各种原因导致异常，还需要在不同高度备份下数据库。修改配置文件中的max_index_height，索引器会跑到这里，检查数据并自动退出，这个时候可以备份数据库。

备份数据库：
cd /data/indexer/db
cp -r mainnet mainnet_xxxxx

xxxx一般填入高度。
备份好了后，继续修改 max_index_height ，继续运行上面的命令跑数据。

建议最后一次备份的数据的区块高度，比现在的高度少12个区块。最后将 max_index_height 设置为0，将 period_flush_to_db 设置为20，再次运行上面的命令，索引器同步到最新高度后，就进入服务状态。同样在浏览器输入 http://127.0.0.1:8009/mainnet/bestheight 查看最新高度。

关闭索引器
先查找索引器的pid，比如 ps -A | grep indexer
然后
kill -2 pid

