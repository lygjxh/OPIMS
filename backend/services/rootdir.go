package services

import "sync"

// RootDir 线程安全地保存"项目文件根目录"这一个可运行时修改的配置值。
// （前身是 FileWatcher，但其文件监控功能从未接线使用，已移除；
//  若将来需要"文件变动自动刷新"，应配合前端 SSE/WebSocket 单独实现。）
type RootDir struct {
	mu   sync.RWMutex
	path string
}

func NewRootDir(path string) *RootDir {
	return &RootDir{path: path}
}

// Get 返回当前根目录。
func (r *RootDir) Get() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.path
}

// Set 更新根目录（用户通过"设置根目录"功能触发）。
func (r *RootDir) Set(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.path = path
}
