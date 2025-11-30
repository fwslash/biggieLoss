package filesystem

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type Dir struct {
	fs          *FS
	Inode       uint64
	Mode        os.FileMode
	ParentInode uint64
	Children    map[string]fs.Node
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.Inode

	a.Mode = os.ModeDir | os.FileMode(0o555)
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	if node, ok := d.Children[name]; ok {
		return node, nil
	}
	return nil, syscall.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	dirs := []fuse.Dirent{
		{Inode: d.Inode, Name: ".", Type: fuse.DT_Dir},
		{Inode: d.ParentInode, Name: "..", Type: fuse.DT_Dir},
	}

	for name, node := range d.Children {
		var a fuse.Attr

		if err := node.Attr(ctx, &a); err != nil {
			log.Printf("Error getting Attr for %s: %v", name, err)
			continue
		}

		dirs = append(dirs, fuse.Dirent{
			Inode: a.Inode,
			Name:  name,
			Type:  fuse.DirentType(a.Mode.Type()),
		})
	}
	return dirs, nil
}

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	if _, exists := d.Children[req.Name]; exists {
		return nil, nil, syscall.EEXIST
	}

	newInode := d.fs.nextInode
	d.fs.nextInode++

	redisKey := redisKeyPrefix + fmt.Sprintf("%d", newInode)

	if err := d.fs.RedisClient.Set(ctx, redisKey, "", 0); err != nil {
		log.Printf("ERROR: Redis SET failed during Create: %v", err)
		return nil, nil, syscall.EIO
	}

	newNode := NewRedisFile(newInode, redisKey, req.Mode, d.fs.RedisClient)

	d.Children[req.Name] = newNode

	resp.Attr.Inode = newInode
	resp.Attr.Mode = req.Mode
	resp.Attr.Size = 0

	return newNode, newNode, nil
}

func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	// 1. Check if the directory already exists in this directory's Children
	if _, exists := d.Children[req.Name]; exists {
		return nil, syscall.EEXIST
	}

	// 2. Generate new Inode and increment the counter
	newInode := d.fs.nextInode
	d.fs.nextInode++

	// 3. Create the new directory node
	newDir := &Dir{
		fs:          d.fs,
		Inode:       newInode,
		Mode:        req.Mode | os.ModeDir,
		Children:    make(map[string]fs.Node),
		ParentInode: d.Inode,
	}

	// 4. Register the new directory in the parent's Children map
	d.Children[req.Name] = newDir

	// 5. Return the new directory node
	return newDir, nil
}

// rm and rmdir implementation
func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	node, ok := d.Children[req.Name]
	if !ok {
		return syscall.ENOENT
	}

	var a fuse.Attr
	if err := node.Attr(ctx, &a); err != nil {
		log.Printf("Error getting Attr for %s: %v", req.Name, err)
		return syscall.EIO
	}

	// Case 1: Deleting a Directory (rmdir)
	if a.Mode&os.ModeDir != 0 {
		if !req.Dir {
			return syscall.EISDIR
		}

		if len(node.(*Dir).Children) > 0 {
			return syscall.ENOTEMPTY
		}

		delete(d.Children, req.Name)
		log.Printf("Removed directory: %s (Inode %d)", req.Name, a.Inode)
		return nil
	}

	redisFile, _ := node.(*RedisFile)

	if err := d.fs.RedisClient.Del(ctx, redisFile.Key); err != nil {
		log.Printf("ERROR: Redis DEL failed for key %s: %v", redisFile.Key, err)
		return syscall.EIO
	}

	delete(d.Children, req.Name)
	log.Printf("Removed persistent file: %s (Inode %d)", req.Name, a.Inode)
	return nil
}
