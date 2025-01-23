# Grapher

![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[中文](README_zh.md)|[English](README.md)

Lightweight in-memory graph database toolkit with storage, algorithms & Cypher-like queries
The entire project is built using Copilot, and the following features are not guaranteed. Please use with caution.

## Features

### Core Modules
- **Graph Storage**  
  ✅ Generic nodes & weighted edges  
  ✅ Thread-safe concurrency  
  ✅ JSON persistence  
  🚧 Versioned snapshots (WIP)

### Query Layer
- **Cypher-like Engine**  
  🚧 Pattern matching & path queries  
  🚧 Built-in indexing
- **OrientSQL Adapter**  
  🚧 SQL-Like matching

### Algorithms
- **Traversal**   
  ✅ BFS/DFS (WIP)  
  🚧 Shortest path

## Quick Start

### Install
```bash
go get github.com/ak-wan/grapher

