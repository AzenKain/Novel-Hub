package services

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/localfs"
)

type SystemLogService interface {
	List(ctx context.Context) ([]*response.LogFileResponse, error)
	Tail(ctx context.Context, name string, lines int, level, search string) (*response.LogTailResponse, error)
	Path(name string) (string, error)
}

type systemLogService struct{ dir string }

func NewSystemLogService(dir string) SystemLogService { return &systemLogService{dir: dir} }

func (s *systemLogService) Path(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || (name != "novelhub.log" && !strings.HasPrefix(name, "novelhub.log.")) {
		return "", apperrors.New(apperrors.ErrBadRequest, "invalid log file name")
	}
	path, err := localfs.SafeJoin(s.dir, name)
	if err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "invalid log file path")
	}
	info, err := os.Lstat(path)
	if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", apperrors.New(apperrors.ErrNotFound, "log file not found")
	}
	return path, nil
}

func (s *systemLogService) List(ctx context.Context) ([]*response.LogFileResponse, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*response.LogFileResponse{}, nil
		}
		return nil, err
	}
	out := make([]*response.LogFileResponse, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || (entry.Name() != "novelhub.log" && !strings.HasPrefix(entry.Name(), "novelhub.log.")) {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			out = append(out, &response.LogFileResponse{Name: entry.Name(), SizeBytes: info.Size(), UpdatedAt: info.ModTime()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *systemLogService) Tail(ctx context.Context, name string, lines int, level, search string) (*response.LogTailResponse, error) {
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	if lines < 1 || lines > 2000 {
		lines = 200
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	level, search = strings.ToLower(level), strings.ToLower(search)
	ring := make([]string, lines)
	count := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		lower := strings.ToLower(line)
		if level != "" && !strings.Contains(lower, `"level":"`+level+`"`) && !strings.Contains(lower, strings.ToUpper(level)) {
			continue
		}
		if search != "" && !strings.Contains(lower, search) {
			continue
		}
		ring[count%lines] = line
		count++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	length := min(count, lines)
	out := make([]string, length)
	start := max(count-lines, 0)
	for i := range length {
		out[i] = ring[(start+i)%lines]
	}
	return &response.LogTailResponse{File: name, Lines: out}, nil
}
