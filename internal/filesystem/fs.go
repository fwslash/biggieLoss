package filesystem

import (
	"context"
	"log"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/fwslash/biggieLoss/internal/redisclient"
)

type FS struct {
	mu          sync.Mutex
	nextInode   uint64
	RedisClient redisclient.RedisService
	RootDir     *Dir
}

func NewFS() *FS {
	redisClient, err := redisclient.NewClient()
	if err != nil {
		log.Fatalf("Could not initialize Redis client: %v", err)
	}

	newFs := &FS{
		RedisClient: redisClient,
		nextInode:   2,
	}

	rootDir := &Dir{
		fs:          newFs,
		Inode:       1,
		Mode:        os.ModeDir | os.FileMode(0o555),
		Children:    make(map[string]fs.Node),
		ParentInode: 1,
	}
	newFs.RootDir = rootDir

	return newFs
}

func (f *FS) Root() (fs.Node, error) {
	return f.RootDir, nil
}

func (f *FS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) (err error) {
	const blockSize = 4096
	const fsBlocks = (1 << 50) / blockSize
	resp.Blocks = fsBlocks  // Total data blocks in file system.
	resp.Bfree = fsBlocks   // Free blocks in file system.
	resp.Bavail = fsBlocks  // Free blocks in file system if you're not root.
	resp.Files = 123        // Total files in file system.
	resp.Ffree = 456        // Free files in file system.
	resp.Bsize = blockSize  // Block size
	resp.Namelen = 255      // Maximum file name length?
	resp.Frsize = blockSize // Fragment size, smallest addressable data size in the file system.
	return nil
}

