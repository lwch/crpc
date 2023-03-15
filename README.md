# crpc

[![crpc](https://github.com/lwch/crpc/actions/workflows/build.yml/badge.svg)](https://github.com/lwch/crpc/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwch/crpc)](https://goreportcard.com/report/github.com/lwch/crpc)
[![license](https://img.shields.io/github/license/lwch/crpc)](https://opensource.org/licenses/MIT)

golang rpc框架，支持以下功能：

1. 流式传输
2. 数据加密
3. 数据压缩
4. 结构序列化

## 分层设计

在crpc框架使用以下的多层设计，每一个层次有其相应的数据结构

    +------------+-------------------+--------------------+-------+
    | data frame | encrypt(optional) | compress(optional) | codec |
    +------------+-------------------+--------------------+-------+

- `data frame`: 数据帧，最底层数据结构，直接面向于tcp协议
- `encrypt`: 数据加密层，目前已支持aes和des加密算法
- `compress`: 数据压缩层，目前已支持gzip和zstd压缩算法
- `codec`: 数据序列化层，目前支持`[]byte`、`http.Request`、`http.Response`三种数据结构的序列化

### 数据帧(network)

数据帧为最基础数据结构，直接作用于tcp链路，其封装格式如下

    +-------------+---------+----------+---------+---------+
    | Sequence(4) | Size(2) | Crc32(4) | Flag(4) | Payload |
    +-------------+---------+----------+---------+---------+

以上内容括号中的数字表示字节数，其中`Flag`字段为枚举类型，枚举值如下

    +---------+------------+----------+---------+---------+---------+-----------+---------------+
    | Open(1) | OpenAck(1) | Close(1) | Data(1) | Ping(1) | Pong(1) | Unused(2) | Stream ID(24) |
    +---------+------------+----------+---------+---------+---------+-----------+---------------+

以上内容括号中的数字表示比特位，其中每一个比特位代表一个标志位，互相之间是互斥关系，目前仅使用了`Flag`字段第一字节的高6位，由于Stream ID字段仅有3字节，因此crpc中仅支持16777215个stream`同时`传输数据

### 数据加密层(encoding/encrypt)

数据加密层用于将原始数据进行加密，在数据加密前会将原始数据的crc32校验码添加到数据尾部作为解密后的校验依据，其封装格式如下：

    +----------+----------+
    + Src Data | Crc32(4) |
    +----------+----------+

- `aes`加密算法: aes加密算法使用256位长度密钥以及16字节的iv进行CBC算法加密
- `des`加密算法: des加密算法使用24字节长度密钥以及8字节的iv进行TripleDES算法加密

当给定密钥长度不足时，底层会重复多次密钥内容以保证加密运算的进行

### 数据编码层(encoding/codec)

数据编码层用于描述原始数据类型，主要作用于数据的序列化和反序列化过程，数据结构如下：

    +---------+---------+
    + Type(1) | Payload |
    +---------+---------+

其中Type字段为1字节，表示当前数据类型，定义如下：

- `0`: 未知数据类型
- `1`: raw data，可反序列化到[]byte
- `2`: http request，可反序列化到http.Request
- `3`: http response，可反序列化到http.Response

#### http请求

grpc框架底层使用`X-Crpc-Request-Id`字段进行request与response的关联，因此在使用过程中请勿使用该字段。

## 示例

TODO