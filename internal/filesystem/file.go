package filesystem

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"github.com/fwslash/biggieLoss/internal/redisclient"
)

// ====================================================================
// RedisFile Structure
// ====================================================================

type RedisFile struct {
	Inode uint64
	Key   string
	Mode  os.FileMode
	Redis redisclient.RedisService
}

func NewRedisFile(inode uint64, key string, mode os.FileMode, svc redisclient.RedisService) *RedisFile {
	return &RedisFile{
		Inode: inode,
		Key:   key,
		Mode:  mode,
		Redis: svc,
	}
}

func (f *RedisFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.Inode
	a.Mode = f.Mode

	content, err := f.Redis.Get(ctx, f.Key)
	if err != nil {
		if err.Error() == fmt.Sprintf("key not found: %s", f.Key) {
			a.Size = 0
			return nil
		}
		log.Printf("ERROR: Redis GET failed during Attr lookup: %v", err)
		return syscall.EIO
	}
	a.Size = uint64(len(content)) * increment
	return nil
}

func (f *RedisFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return f, nil
}

func (f *RedisFile) ReadAll(ctx context.Context) ([]byte, error) {
	content, err := f.Redis.Get(ctx, f.Key)
	if err != nil {
		if err.Error() == fmt.Sprintf("key not found: %s", f.Key) {
			return nil, syscall.ENOENT
		}
		log.Printf("ERROR: Redis GET failed during ReadAll: %v", err)
		return nil, syscall.EIO
	}
	return []byte(content), nil
}

func (f *RedisFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	err := f.Redis.SetRange(
		ctx,
		f.Key,
		int64(req.Offset),
		string(req.Data),
	)

	if err != nil {
		log.Printf("ERROR: Redis SETRANGE failed during Write: %v", err)
		return syscall.EIO
	}

	resp.Size = len(req.Data)
	return nil
}
