# Image Gallery — veil.ortlinde.com

基于 [veil.ortlinde.com](https://veil.ortlinde.com) 接口开发的看图网站，使用 Go + Vue 3 + TailwindCSS，单二进制部署。

## 目录结构

```
├── main.go              # Go HTTP 服务器（反向代理 + 缓存 + 密码验证）
├── go.mod               # Go 模块
├── deploy.sh            # 一键部署脚本
├── Dockerfile           # 多阶段构建（node → golang → alpine）
├── frontend/            # Vue 3 SPA 前端
│   ├── src/
│   │   ├── App.vue      # 主页面组件（含密码弹窗）
│   │   ├── main.js      # Vue 入口
│   │   └── style.css    # TailwindCSS
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
└── tests/
    └── gallery.test.js  # Playwright 端到端测试
```

## 功能

| 功能 | 说明 |
|------|------|
| 访问密码 | 部署时通过 `ACCESS_PASSWORD` 环境变量设置，进入首页先弹窗验证 |
| 标签浏览 | 搜索标签（自动补全下拉 + 标签数量）、点击精选标签/分类 |
| 方向筛选 | 竖/横/All 三态切换，带图标 |
| 随机模式 | 每次加载 1 张图片，支持滚动无限加载 / Load More 按钮 |
| 标签预览 | 选中标签展示 6 张预览图 + Load More 继续加载 |
| 弹窗查看 | 全屏预览 + 元数据面板（ID、尺寸、方向、标签） |
| 图片缩放 | Ctrl+滚轮（桌面）、双指捏合（移动）、双击切换 1×↔2.5× |
| 图片平移 | 缩放后拖拽移动 |
| 内存缓存 | tags/featured-tags/categories 缓存 120s，元数据 300s |
| 请求日志 | 每条代理请求记录 method、path、status、耗时 |

## 快速部署

### 使用部署脚本（推荐）

```bash
git clone https://github.com/jingyuan9527/go-images1.git
cd go-images1

# 基本使用（无密码，内网访问）
./deploy.sh

# 设置访问密码
ACCESS_PASSWORD=your_secret ./deploy.sh

# 开放局域网访问
BIND_ADDR=0.0.0.0 ./deploy.sh

# 指定端口
PORT=3000 ./deploy.sh

# 多参数组合使用
ACCESS_PASSWORD=your_secret BIND_ADDR=0.0.0.0 PORT=3000 ./deploy.sh
ACCESS_PASSWORD=your_secret HTTP_PROXY=http://your-proxy:port ./deploy.sh
```

### 手动 Docker

```bash
docker build -t go-images .
docker run -d \
  -p 127.0.0.1:8808:8808 \
  -e ACCESS_PASSWORD=your_secret \
  -e HTTP_PROXY=http://your-proxy:port \
  -e HTTPS_PROXY=http://your-proxy:port \
  --name go-images \
  --restart unless-stopped \
  go-images
```

### 本地编译

```bash
npm --prefix frontend install && npm --prefix frontend run build
go build -o webapp .
ACCESS_PASSWORD=your_secret ./webapp
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `PORT` | `8808` | 服务监听端口 |
| `ACCESS_PASSWORD` | (空) | 访问密码，不设置则跳过验证 |
| `BIND_ADDR` | `127.0.0.1` | 绑定地址（仅 deploy.sh） |
| `HTTP_PROXY` | (空) | HTTP 代理地址 |
| `HTTPS_PROXY` | (空) | HTTPS 代理地址 |

## 代理说明

上游接口 `veil.ortlinde.com` 受 Cloudflare 保护。在以下场景可能需要配置代理：

- **国内服务器**直连被限流（返回 403）
- VM 本身配置系统代理，但 Docker 容器不继承

服务端已设置浏览器类请求头（User-Agent、Referer 等）模拟正常访问。如仍遇 403，通过 `HTTP_PROXY`/`HTTPS_PROXY` 环境变量指定代理即可。

## API 反向代理

| 前端路径 | 上游地址 | 说明 |
|----------|----------|------|
| `GET /api/proxy/v1/random` | `GET /v1/random` | 随机图片流 |
| `GET /api/proxy/v1/tag/{name}/preview` | `GET /v1/tag/{name}/preview` | 标签预览 6 张 |
| `GET /api/proxy/v1/image/{id}` | `GET /v1/image/{id}` | 具体图片 |
| `GET /api/proxy/v1/image/{id}/meta` | `GET /v1/image/{id}/meta` | 图片元数据 |
| `GET /api/proxy/v1/tags` | `GET /v1/tags` | 所有标签 |
| `GET /api/proxy/v1/featured-tags` | `GET /v1/featured-tags` | 精选标签 |
| `GET /api/proxy/v1/categories` | `GET /v1/categories` | 分类列表 |

## 运行测试

```bash
npm --prefix tests install
npx playwright install chromium
# 部署时需设置 ACCESS_PASSWORD=test123（测试写死的密码）
node --test tests/gallery.test.js
```

测试覆盖：首页加载、标签渲染、分类按钮、方向筛选、图片搜索、弹窗展示、双击缩放、Load More。