# Grapher

![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[中文](README_zh.md)|[English](README.md)

轻量化内存图数据库基础组件，提供图存储、遍历算法和类Cypher/OrientSQL查询引擎

## 功能特性

### 核心模块
- **图存储引擎**  
  ✅ 支持泛型节点与带权边  
  ✅ 线程安全并发控制  
  ✅ JSON持久化与恢复  
  🚧 图版本快照（开发中）

### 查询层
- **Cypher-like引擎**  
  🚧 支持模式匹配与路径查询  
  🚧 内置索引加速
- **OrientSQL适配器**  
  🚧 类SQL的图遍历语法支持

### 算法库
- **基础遍历**  
  ✅ BFS/DFS (开发中)  
  🚧 最短路径算法
- **高级算法**  
  🚧 环路检测

## 快速开始

### 安装
```bash
go get github.com/ak-wan/grapher

