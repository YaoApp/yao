<p align="center">
    <h1 align="center">YAO App Engine</h1>
</p>

<p align="center">
  <a aria-label="website" href="https://yaoapps.com" target="_blank">
    Website
  </a>
  ·
  <a aria-label="producthunt" href="https://www.producthunt.com/posts/yao-app-engine" target="_blank">
    Producthunt
  </a>
  ·
  <a aria-label="twitter" href="https://twitter.com/YaoApp" target="_blank">
    Twitter
  </a>
  <!-- ·
  <a aria-label="discord" href="https://discord.gg/nsKmCXwvxU" target="_blank">
    Discord
  </a> -->
</p>

<p align="center">
  <a aria-label="UnitTest" href="https://github.com/YaoApp/yao/actions/workflows/unit-test.yml" target="_blank">
    <img src="https://github.com/YaoApp/yao/actions/workflows/unit-test.yml/badge.svg">
  </a>
  <a aria-label="codecov" href="https://codecov.io/gh/YaoApp/yao" target="_blank">
    <img src="https://codecov.io/gh/YaoApp/yao/branch/main/graph/badge.svg?token=294Y05U71J">
  </a>
  <a aria-label="Go Report Card" href="https://goreportcard.com/report/github.com/yaoapp/yao" target="_blank">
    <img src="https://goreportcard.com/badge/github.com/yaoapp/yao">
  </a>
  <a aria-label="Go Reference" href="https://pkg.go.dev/github.com/yaoapp/yao" target="_blank">
    <img src="https://pkg.go.dev/badge/github.com/yaoapp/yao.svg">
  </a>
  <a href="https://app.fossa.com/projects/git%2Bgithub.com%2FYaoApp%2Fyao?ref=badge_shield" alt="FOSSA Status">
    <img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2FYaoApp%2Fyao.svg?type=shield"/>
  </a>
</p>

https://github.com/YaoApp/yao/assets/1842210/6b23ac89-ef6e-4c24-874f-753a98370dec

[中文介绍](README.zh-CN.md)

YAO is an open-source application engine, written in Golang, in the form of a command-line tool that can be downloaded and used immediately. It is suitable for developing business systems, website/APP API, admin panel, self-built low-code platforms, etc.

YAO adopts a flow-based programming model to implement various functions by writing YAO DSL (Logical Description in JSON format) or using JavaScript to write processes. The YAO DSL can be written in several ways:

1. Purely hand-written

2. Use automated scripts to generate contextual logic

3. Use the visual editor to create by "drag and drop"

**Documentation:** https://yaoapps.com/en-US/doc

## Latest Version download and installation

https://github.com/YaoApp/xgen-dev-app

## Demo

Applications developed with YAO

| Application          | Description                                          | Repository                              |
| -------------------- | ---------------------------------------------------- | --------------------------------------- |
| yaoapp/yao-examples  | Yao Examples                                         | https://github.com/YaoApp/yao-examples  |
| yaoapp/yao-knowledge | A knowledge base application                         | https://github.com/YaoApp/yao-knowledge |
| yaoapp/xgen-dev-app  | A demo application                                   | https://github.com/YaoApp/xgen-dev-app  |
| yaoapp/demo-project  | A demo application for project management            | https://github.com/yaoapp/demo-project  |
| yaoapp/demo-finance  | A demo application for financial management          | https://github.com/yaoapp/demo-finance  |
| yaoapp/demo-plm      | A demo application for production project management | https://github.com/yaoapp/demo-plm      |

## Introduce

**Yao allows developers to create web services by processes.** Yao is a app engine that creates a database model, writes API services, and describes dashboard interface just by JSON for web & hardware, and 10x productivity.

Yao is based on the **flow-based** programming idea, developed in the **Go** language, and supports multiple ways to expand the data stream processor. This makes Yao extremely versatile, which can replace programming languages ​​in most scenarios, and is 10 times more efficient than traditional programming languages ​​in terms of reusability and coding efficiency; application performance and resource ratio Better than **PHP**, **JAVA** and other languages.

Yao has a built-in data management system. By writing **JSON** to describe the interface layout, 90% of the common interface interaction functions can be realized. It is especially suitable for quickly making various management background, CRM, ERP and other internal enterprise systems. Special interactive functions can also be implemented by writing extension components or HTML pages. The built-in management system is not coupled with Yao, and any front-end technologies such as **VUE** and **React** can be used to implement the management interface.

## Install

Run the script under terminal: (MacOS/Linux)

```bash
curl -fsSL https://website.yaoapps.com/install.sh | bash
```

For Windows users, please refer to the Installation and Debugging chapter: [Installation and debugging](https://yaoapps.com/en-US/doc/Introduction/Install)

## Getting Started

See [documentation](https://yaoapps.com/en-US/doc/Introduction/Getting%20Started) for more details.

### Create a blank project

Create a new application directory, enter the application directory, run the `yao start` command, and start the installation.

```bash
mkdir -p /data/app # create project directory
cd /data/app # Enter the project directory
yao start # Start installation
```

**Default Account**

- User: **xiang@iqka.com**

- Password: **A123456p+**

### Download a project

Create a new project directory, enter the project directory, run the `yao get` command, and download the application code.

```bash
mkdir -p /data/app # create project directory
cd /data/app # Enter the project directory
yao get yaoapp/demo-plm # download demo-plm
yao start # Start installation
```

**Default Account**

- User: **xiang@iqka.com**

- Password: **A123456p+**

## About Yao

Yao's name is derived from the Chinese character **爻 (yáo)**, the basic symbol that makes up the Eight Trigrams. The Eight Trigrams is a symbol system created by the ancient god Fuxi after observing and summarizing the laws of nature, which can refer to everything. Yao has two states of yin and yang, like 0 and 1. The transformation of yin and yang of Yao drives the replacement of Eight Trigrams, so as to summarize and record the development law of things.
