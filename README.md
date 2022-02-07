# Yao

[![UnitTest](https://github.com/YaoApp/yao/actions/workflows/unit-test.yml/badge.svg)](https://github.com/YaoApp/yao/actions/workflows/unit-test.yml)
[![codecov](https://codecov.io/gh/YaoApp/yao/branch/main/graph/badge.svg?token=294Y05U71J)](https://codecov.io/gh/YaoApp/yao)

Yao 是一款 **Go** 语言驱动的低代码应用引擎，通过编写 JSON 描述即可快速制作 **API 接口**，**数据管理系统** ，**命令行工具** 等应用程序，应用可以运行在本地、云端和物联网设备上。

## 介绍

Yao 基于 **flow-based** 编程思想，采用 **Go** 语言开发，支持多种方式扩展数据流处理器。这使得 Yao 具有极好的**通用性**，大部分场景下可以代替编程语言, 在复用性和编码效率上是传统编程语言的 **10 倍**；应用性能和资源占比上优于 **PHP**, **JAVA** 等语言。

Yao 内置了一套数据管理系统，通过编写 **JSON** 描述界面布局，即可实现 90% 常见界面交互功能，特别适合快速制作各类管理后台、CRM、ERP 等企业内部系统。对于特殊交互功能亦可通过编写扩展组件或 HTML 页面的方式实现。内置管理系统与 Yao 并不耦合，亦可采用 **VUE**, **React** 等任意前端技术实现管理界面。

Yao 的名字源于汉字**爻(yáo)**，是构成八卦的基本符号。八卦，是上古大神伏羲观测总结自然规律后，创造的一个可以指代万事万物的符号体系。爻，有阴阳两种状态，就像 0 和 1。爻的阴阳转换，驱动八卦更替，以此来总结记录事物的发展规律。

## 演示

**客户关系管理系统** [源码](https://github.com/yaoapp/crm) [演示](https://demo-crm.yaoapps.com)

一套通用 CRM 管理系统

**智能仓库管理系统** [源码](https://github.com/yaoapp/warehouse) [演示](https://demo-warehouse.yaoapps.com)

云+边物联网应用示例，支持人脸识别、RFID 的无人值守智能仓库管理系统。

## 起步

**注意：开始前需要了解 JSON、RESTFulAPI、关系型数据库的基本概念和常识，并可以使用常见终端命令。如需处理非常复杂的业务逻辑，则需要掌握 JavaScript 语言。**

我们准备了一个在线开发调试环境，帮助您快速上手体验。**在线开发环境仅做体验使用， 数据每小时重置** , 在实际开发中，请在本地安装 Yao 进行开发调试，具体参考 [安装调试](/a.介绍/b.安装调试.mdx) 章节的文档。如果您的应用开发完成，需要在生产环境部署，请参考 [部署](/e.部署.mdx) 章节文档。

<Extend
title="打开在线调试环境"
desc="在线开发测试环境仅做体验使用，数据每小时重置"
link="https://ide-demo.yaoapps.com"

> </Extend>

### 第一步: 创建项目

新建一个项目目录，进入项目目录，运行 `yao init` 命令，创建一个空白的 Yao 应用。

```bash
mkdir -p /data/crm  # 创建项目目录
cd /data/crm  # 进入项目目录
yao init # 运行初始化程序
```

命令运行成功后，将创建 `app.json文件` , `db目录`, `ui目录` 和 `data目录`

```bash
├── data        # 用于存放应用产生的文件，如图片,PDF等
├── db          # 用于存放 SQLite 数据库文件
│   └── xiang.db
└── ui          # 静态文件服务器文件目录，可以放置自定义前端制品，该目录下文件可通过 http://host:port/文件名称 访问。
└── app.json    # 应用配置文件, 用来定义应用名称等
```

### 第二步: 创建数据模型

设计一张数据表 customer，用于存放客户资料数据。数据表结构如下:

| 字段       | 类型       | 长度/参数                  | 默认值  | 说明                      |
| ---------- | ---------- | -------------------------- | ------- | ------------------------- |
| id         | ID         |                            |         | 数据表 ID                 |
| channel_id | bigInteger |                            |         | 客户来源                  |
| name       | string     | 80                         |         | 客户姓名                  |
| company    | string     | 200                        |         | 公司名称                  |
| gender     | integer    |                            | 0       | 性别 0, 未知，1 男，2，女 |
| birthday   | date       |                            |         | 生日                      |
| desc       | text       |                            |         | 简历                      |
| status     | enum       | ["enabled"], ["disabled"], | enabled | 状态                      |

创建数据表对应的 `JSON 描述文件` :

在项目 `models` 目录下 (例如: `/data/crm/models`)，创建数据模型描述文件 `customer.mod.json`

```json
{
  "name": "客户",
  "table": { "name": "customer", "comment": "客户表" },
  "columns": [
    { "label": "ID", "name": "id", "type": "ID", "comment": "ID" },
    {
      "label": "来源",
      "name": "channel_id",
      "type": "bigInteger",
      "comment": "客户来源",
      "nullable": true,
      "index": true
    },
    {
      "label": "姓名",
      "name": "name",
      "type": "string",
      "length": 80,
      "comment": "客户姓名",
      "nullable": true,
      "index": true
    },
    {
      "label": "公司",
      "name": "company",
      "type": "string",
      "length": 200,
      "comment": "公司全称",
      "nullable": true,
      "index": true
    },
    {
      "label": "性别",
      "name": "gender",
      "type": "integer",
      "comment": "性别",
      "default": 0,
      "index": true
    },
    {
      "label": "生日",
      "name": "birthday",
      "type": "date",
      "comment": "性别",
      "nullable": true,
      "index": true
    },
    {
      "label": "介绍",
      "name": "desc",
      "type": "text",
      "comment": "介绍",
      "nullable": true
    },
    {
      "label": "状态",
      "name": "status",
      "type": "enum",
      "default": "enabled",
      "option": ["enabled", "disabled"],
      "comment": "状态 enabled 开启, disabled 关闭",
      "index": true
    }
  ],
  "option": { "timestamps": true, "soft_deletes": true },
  "values": [{ "id": 1, "name": "莉莉", "company": "yaoapps.com" }]
}
```

使用 `yao migrate` 命令创建数据表，打开命令行终端，**在项目根录下运行**:

```bash
yao migrate
```

**技巧: 调试过程中，需要频繁检查数据。可以使用 `yao run` 命令查询数据。例如:在命令行终端中运行 `yao run models.customer.find 1` 即可查询 `customer` 数据表 ID=1 的数据。**

### 第三步: 编写管理界面

在项目 `tables` 目录下 (例如: `/data/crm/models`)，创建数据表格描述文件 `customer.tab.json`

```json
{
  "name": "客户",
  "version": "1.0.0",
  "decription": "客户管理数据表格",
  "bind": {
    "model": "customer"
  },
  "columns": {},
  "filters": {
    "关键词": { "@": "filter.关键词", "in": ["where.name.match"] },
    "排序": { "@": "filter.排序" }
  },
  "list": {
    "primary": "id",
    "layout": {
      "columns": [
        { "name": "ID", "width": 80 },
        { "name": "公司", "width": 200 },
        { "name": "姓名", "width": 100 },
        { "name": "性别", "width": 80 },
        { "name": "生日", "width": 100 },
        { "name": "来源", "width": 80 },
        { "name": "状态", "width": 140 },
        { "name": "更新时间", "width": 160 }
      ],
      "filters": [{ "name": "关键词" }, { "name": "来源", "width": 3 }]
    },
    "actions": {
      "create": {
        "type": "button",
        "props": { "label": "添加客户", "icon": "fas fa-plus" }
      },
      "pagination": {}
    }
  },
  "edit": {
    "primary": "id",
    "layout": {
      "fieldset": [
        {
          "columns": [
            { "name": "姓名", "width": 12 },
            { "name": "性别", "width": 3 },
            { "name": "生日", "width": 3 },
            { "name": "来源", "width": 3 },
            { "name": "状态", "width": 3 },
            { "name": "公司", "width": 24 },
            { "name": "介绍", "width": 24 }
          ]
        }
      ]
    },
    "actions": {
      "cancel": {},
      "save": { "@": "action.保存" },
      "delete": { "@": "action.删除" }
    }
  }
}
```

**注意：由于该项目诞生之初主要是为了提高团队内部生产力，在约定俗成的写法描述下，并没有做细粒度的抛错处理，所以开发者可能会在编写 JSON 调试界面的过程当中遇到界面为空的情况。处置方式请查阅组件文档。**

**启动服务**

打开命令行终端，**在项目根录下运行**:

```bash
yao start
```

1. 打开浏览器, 访问 `https://127.0.0.1:5099/xiang/login/admin`，

2. 输入默认用户名: `xiang@iqka.com`， 密码: `A123456p+`

3. 成功登录后，在地址栏输入 `https://127.0.0.1:5099/xiang/table/customer`

4. 建议将 `/xiang/table/customer` 路由添加为菜单项。
