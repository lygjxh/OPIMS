package services

import (
	"path/filepath"
	"sync"
)

// 根目录（`海外运营中心`）下的固定子目录名。2026-07-26：根目录上移一层，
// 项目文件不再直接位于根目录下，而是位于 `<根>/01.Project Files/`。
const (
	ProjectFilesSubdir = "01.Project Files"
	QualDocsSubdir     = "07.Sub-contractor/01.Qualification Documents"
	// PolicySubdir 指向政策库根（同时也是一个 Obsidian vault 根，含 .obsidian 配置目录）。
	// 指向 08.Policy 而非直接指向其下的「出入境政策」，是为了后续新增
	// 税务政策、劳工法规等分类时可直接复用，无需再改本文件。
	PolicySubdir = "08.Policy"
	// EntryExitPolicyCategory 是政策库下「出入境政策」分类的目录名。
	EntryExitPolicyCategory = "出入境政策"
)

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

// ProjectsDir 返回项目文件所在目录：`<根>/01.Project Files`。
func (r *RootDir) ProjectsDir() string {
	return filepath.Join(r.Get(), ProjectFilesSubdir)
}

// QualDir 返回分包商资质文件目录：`<根>/07.Sub-contractor/01.Qualification Documents`。
func (r *RootDir) QualDir() string {
	return filepath.Join(r.Get(), filepath.FromSlash(QualDocsSubdir))
}

// PolicyDir 返回政策库根目录：`<根>/08.Policy`。
func (r *RootDir) PolicyDir() string {
	return filepath.Join(r.Get(), filepath.FromSlash(PolicySubdir))
}

// EntryExitPolicyDir 返回出入境政策目录：`<根>/08.Policy/出入境政策`。
func (r *RootDir) EntryExitPolicyDir() string {
	return filepath.Join(r.PolicyDir(), EntryExitPolicyCategory)
}
