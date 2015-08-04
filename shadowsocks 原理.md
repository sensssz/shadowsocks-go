# shadowsocks 原理

### shadowsocks
　　最近在学习 Go 语言，学完之后手痒痒的，忍不住就想用它来做些东西。shadowsocks 是一个我觉得蛮好的 [Kata](http://codekata.com/)，一直很想自己实现看看，现在趁着这个机会也一并做了。
　　要实现 shadowsocks 就得先了解它的原理。我在网上找了些文章看了看，其实还是很简单的。shadowsocks 的工作原理用图像表示出来就是这个样子<sup>[1](http://www.cnblogs.com/foreverfree/p/3771531.html)</sup>

        PC －－> local client －－> remote server －－> website
                | -------- shadowsocks -------- |

　　中间的 local client 和 remote server 就是 shadowsocks，而它的作用其实就是实现了一个 SOCKS5 的服务器。SOCKS5 是一个代理协议，当一个台 PC 使用一个正常的 SOCKS5 服务器作为代理服务器的时候，应该是这样子的

        PC －－> SOCKS5 server －－> anywhere

　　 但是 shadowsocks 将它分为了两个部分。PC 将 shadowsocks 的 local client 当做一个 SOCKS5 server，将数据发送给 local client，local client 将数据包解开，用一个私有的请求格式，通过 local client 的配置文件里面指定的加密算法加密后传给 remote server，remote server 解密后处理 SOCKS 请求，然后将结果返回给 local client。整个过程非常简单明了。shadowsocks 将 SOCKS5 server 分为两个部分的意义在于在中间加了一个加密层，把从 PC 发送出去的 SOCKS 报文进行加密，增加隐蔽性。

### SOCKS5
　　了解了 shadowsocks 的原理之后，接下来就是如何实现 SOCKS5 协议了。SOCKS5 协议的 specification 在 [RFC 1928](https://www.ietf.org/rfc/rfc1928.txt)。SOCKS5 同时支持 TPC 和 UDP 两种协议。以下都是基于 TCP 的协议内容。在 TCP 下，客户端和服务器端的握手如下：

        client －－－－－－－－－－－－－－－－－－－－－ server
	       |   －－－－－－－ greeting －－－－－－－>   |
	       |   <－－－－－－ response －－－－－－－－   |
	       |   <－－－－ [method specific] －－－－>   |
	       |   －－－－－ connect request －－－－－>   |
	       |   <－－－－ connect response －－－－－   |
	       
连接建立成功后，客户端就可以把报文发送到服务器端，并接受从服务器端返回的数据了。

#### SOCKS 报文格式
1. `greeting`

                   +----+----------+----------+
                   |VER | NMETHODS | METHODS  |
                   +----+----------+----------+
                   | 1  |    1     | 1 to 255 |
                   +----+----------+----------+                   
这里的 `VER` 是指 `version`，SOCKS5 里面此处的值是 0x05，`NMETHODS` 代表 `METHODS` 里面列举的 methods 的数量。目前支持的 `METHOD` 有：

          o  0x00 NO AUTHENTICATION REQUIRED
          o  0x01 GSSAPI
          o  0x02 USERNAME/PASSWORD
          o  0x03 to 0x7F IANA ASSIGNED
          o  0x80 to 0xFE RESERVED FOR PRIVATE METHODS
          o  0xFF NO ACCEPTABLE METHODS
shadowsocks 里所选择的 method 是 0x00。
2. `greeting response`

                         +----+--------+
                         |VER | METHOD |
                         +----+--------+
                         | 1  |   1    |
                         +----+--------+
此处的 `VER` 的意义和上面相同， `METHOD` 代表所选择的 method，如果值是 0xFF 的话，代表 `greeting` 里面所列举的 method 都不被支持。此时按照协议客户端**必须**关闭连接。
3. `connect request`

        +----+-----+-------+------+----------+----------+
        |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
        +----+-----+-------+------+----------+----------+
        | 1  |  1  | X'00' |  1   | Variable |    2     |
        +----+-----+-------+------+----------+----------+
`connect request` 里包含了所要连接的主机（不是指代理服务器，而是真正想要访问的网站）的信息。这个报文里面各项的意义分别是

          o  VER 协议版本，值为 0x05
          o  CMD
             o  CONNECT: 0x01
             o  BIND: 0x02
             o  UDP ASSOCIATE: 0x03
          o  RSV 保留字段
          o  ATYP 目标地址的类型，是下面三种的其中一种
             o  IPV4 地址: 0x01
             o  域名: 0x03
             o  IPV6 地址: 0x04
          o  DST.ADDR 目标地址
          o  DST.PORT 目标端口
`DST.ADDR` 的长度取决于地址类型。如果是 `IPV4` 的话，长度为 4 个字节，`IPV6` 的话，则是 16 个字节。如果地址类型是一个域名的话，那么第一个字节代表域名的字节数，紧接着就是域名字符串，结尾没有 `\0` 字符。
4. `connect response`

        +----+-----+-------+------+----------+----------+
        |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
        +----+-----+-------+------+----------+----------+
        | 1  |  1  | X'00' |  1   | Variable |    2     |
        +----+-----+-------+------+----------+----------+
接收到一个 `connect request` 后，服务器会根据报文中的信息连接相应的服务器，并根据连接的结果返回一个 `response` 报文。报文各项的意义是

          o  VER 协议版本，值为 0x05
          o  REP 代表连接的结果
             o  0x00 连接成功
             o  0x01 SOCKS 服务器错误
             o  0x02 现有的规则不允许的连接
             o  0x03 网络不可达
             o  0x04 主机不可达
             o  0x05 连接被拒绝
             o  0x06 TTL（跳数）超时
             o  0x07 不支持的命令
             o  0x08 不支持的地址类型
             o  0x09-0xFF 未被使用
          o  RSV 保留未使用
          o  ATYP 地址类型
             o  IPV4 地址: 0x01
             o  域名: 0x03
             o  IPV6 地址: 0x04
          o  BND.ADDR 服务器绑定的地址
          o  BND.PORT 服务器绑定的端口
如果这个报文种 `REP` 的值代表某个错误的话，按照协议，服务器端**必须**在发送这个报文之后，在发现错误出现的 **10 秒**内关闭连接。

> Written with [StackEdit](https://stackedit.io/).