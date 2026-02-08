索引器安装说明
====
自己拉取代码，并且本地编译，得到 indexer-mainnet 和 indexer-testnet


在自己的数据区创建一个目录，比如 /data/indexer
将编译好的indexer和两个conf.yaml文件拷贝到这个目录下

修改conf.yaml，主要修改btc的用户名和密码，还有pubkey，将satsnet.conf中的miningpubkey复制过来就行。

最后，执行下面命令，开始跑主网数据。
nohup ./indexer_mainnet -env ./conf_mainnet.yaml > ./nohup_mainnet.log 2>&1 &

测试网：
nohup ./indexer_testnet -env ./conf_testnet.yaml > ./nohup_testnet.log 2>&1 &


测试网数据可能几个小时就完成，然后自动进入服务状态。可以打开浏览器，输入：http://127.0.0.1:8019/btc/testnet/bestheight
查看索引器同步的最新高度。

主网数据一般要一周。为了确保数据不会因为各种原因导致异常，还需要在不同高度备份下数据库。修改配置文件中的max_index_height，索引器会跑到这里，检查数据并自动退出，这个时候可以备份数据库。

备份数据库：
cd /data/indexer/db
cp -r mainnet mainnet_xxxxx

xxxx一般填入高度。
备份好了后，继续修改 max_index_height ，继续运行上面的命令跑数据。

建议最后一次备份的数据的区块高度，比现在的高度少12个区块。最后将 max_index_height 设置为0，将 period_flush_to_db 设置为20，再次运行上面的命令，索引器同步到最新高度后，就进入服务状态。同样在浏览器输入 http://127.0.0.1:8009/btc/mainnet/bestheight 查看最新高度。

关闭索引器
不要强制关闭索引器，可能会破坏数据库。
先查找索引器的pid，比如 ps -A | grep indexer
然后
kill -2 pid

