// backend 模拟真实后端：configx + logx + eventx + httpx + dix + dbx(SQLite)
//
// 运行: go run ./backend
// 环境变量: APP_SERVER_PORT=3000, APP_DB_DSN=file:app.db
package main

import "github.com/arcgolabs/dix/examples/backend/app"

func main() {
	app.Run()
}
