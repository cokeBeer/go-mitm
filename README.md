# go-mitm
## Introduction
最近在研究怎么用go语言写一个mitm proxy，参考了[mitm](https://github.com/lyyyuna/mitm)这个项目。实际使用的过程中发现现在谷歌浏览器会报INVALI_COMMON_NAME错误，而使用[proxify](https://github.com/projectdiscovery/proxify)代理是正常的。怀疑是证书生成模块的问题。我使用后者的证书生成模块代替了原来的证书生成模块，成功了修复了报错。
## Install
```
git clone https://github.com/cokeBeer/go-mitm
cd go-mitm
```
## Usage
执行如下指令会在8080端口开启代理
```
go run main.go --port 8080 --log ./mitm.log
```
同时文件夹下会生成
```
cacert.pem
```
将这个证书导入浏览器，设置为始终信任即可
## How it work
见下图
![img](https://i.bmp.ovh/imgs/2022/04/11/86f40bf749ef1467.png)
## Thanks to
[mitm](https://github.com/lyyyuna/mitm)\
[proxify](https://github.com/projectdiscovery/proxify)