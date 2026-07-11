package setting

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	sandboxv2 "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/runtime"
	taitypes "github.com/yaoapp/yao/tai/types"
)

const sandboxRegistryNS = "sandbox.registry"

// pullState tracks an in-progress image pull operation.
type pullState struct {
	ImageRef string
	NodeID   string
	Progress int    // 0-100
	Error    string // non-empty on failure
	Done     bool
}

var pullTracker sync.Map // key: "nodeID:imageRef"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func imageRefToID(ref string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

func idToImageRef(id string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	return string(b), err
}

func friendlyImageError(locale string, msg string) string {
	isCN := strings.HasPrefix(strings.ToLower(locale), "zh")

	if strings.Contains(msg, "conflict") || strings.Contains(msg, "must force") {
		if isCN {
			return "该镜像正在被运行中的沙箱使用，请先停止相关沙箱后再删除"
		}
		return "This image is in use by a running sandbox. Please stop the sandbox first before removing."
	}
	if strings.Contains(msg, "No such image") || strings.Contains(msg, "not found") {
		if isCN {
			return "镜像不存在或已被删除"
		}
		return "Image not found or already removed"
	}
	if strings.Contains(msg, "no matching manifest") {
		if isCN {
			return "该镜像不支持当前系统架构（" + msg + "）"
		}
		return "This image does not support the current architecture (" + msg + ")"
	}
	if strings.Contains(msg, "pull access denied") || strings.Contains(msg, "repository does not exist") {
		if isCN {
			return "镜像不存在或无拉取权限，请检查镜像名称和仓库配置"
		}
		return "Image not found or access denied. Please check the image name and registry config."
	}
	if strings.Contains(msg, "dial tcp") || strings.Contains(msg, "timeout") || strings.Contains(msg, "TLS handshake") {
		if isCN {
			return "无法连接镜像仓库，请检查网络连接"
		}
		return "Cannot connect to the image registry. Please check your network."
	}
	if isCN {
		return "操作失败: " + msg
	}
	return "Operation failed: " + msg
}

func friendlyOS(goos string) string {
	switch strings.ToLower(goos) {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		return goos
	}
}

func getSandboxManager() *sandboxv2.Manager {
	defer func() { recover() }()
	return sandboxv2.M()
}

func sandboxNodeOwnedBy(snap *taitypes.NodeMeta, authInfo *oauthTypes.AuthorizedInfo) bool {
	if authInfo == nil {
		return true
	}
	if authInfo.TeamID != "" {
		return snap.Auth.TeamID == authInfo.TeamID
	}
	if authInfo.UserID != "" {
		return snap.Auth.TeamID == "" && snap.Auth.UserID == authInfo.UserID
	}
	return true
}

type dockerInfoResult struct {
	Version  string
	MemTotal int64
	NCPU     int
}

func fetchDockerInfo(nodeID string) *dockerInfoResult {
	res, ok := tai.GetResources(nodeID)
	if !ok || res.Runtime == nil {
		return nil
	}
	cli := runtime.DockerCli(res.Runtime)
	if cli == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info, err := cli.Info(ctx)
	if err != nil {
		return nil
	}
	return &dockerInfoResult{
		Version:  info.ServerVersion,
		MemTotal: info.MemTotal,
		NCPU:     info.NCPU,
	}
}

// collectAssistantImages traverses assistant cache to find sandbox image requirements.
// Returns map[imageRef][]assistantDisplayName (locale-resolved).
func collectAssistantImages(locale string) map[string][]string {
	cache := assistant.GetCache()
	if cache == nil {
		return nil
	}
	result := make(map[string][]string)
	for _, ast := range cache.All() {
		if ast == nil {
			continue
		}
		var imageRef string
		if ast.SandboxV2 != nil && ast.SandboxV2.Computer.Image != "" {
			imageRef = ast.SandboxV2.Computer.Image
		} else if ast.Sandbox != nil && ast.Sandbox.Image != "" {
			imageRef = ast.Sandbox.Image
		}
		if imageRef != "" {
			name := ast.GetName(locale)
			if name == "" {
				name = ast.ID
			}
			result[imageRef] = append(result[imageRef], name)
		}
	}
	return result
}

// splitImageRef splits "repo/name:tag" into (imageName, tag).
func splitImageRef(ref string) (string, string) {
	if idx := strings.LastIndex(ref, ":"); idx > 0 && !strings.Contains(ref[idx:], "/") {
		return ref[:idx], ref[idx+1:]
	}
	return ref, "latest"
}

// getNodeResources retrieves ConnResources for a node with image capability.
// Returns (resources, httpStatus, errorMessage).
func getNodeResources(nodeID string) (*tai.ConnResources, int, string) {
	reg := registry.Global()
	if reg == nil {
		return nil, http.StatusServiceUnavailable, "tai registry not initialized"
	}
	meta, ok := reg.Get(nodeID)
	if !ok {
		return nil, http.StatusNotFound, "node not found: " + nodeID
	}
	if meta.Status != "online" {
		return nil, http.StatusBadRequest, "node is offline: " + nodeID
	}
	res, ok := tai.GetResources(nodeID)
	if !ok {
		return nil, http.StatusBadGateway, "cannot reach node: " + nodeID
	}
	if res.Image == nil {
		return nil, http.StatusBadRequest, "Docker not available on this node"
	}
	return res, 0, ""
}

// ---------------------------------------------------------------------------
// GET /setting/sandbox
// ---------------------------------------------------------------------------

func handleSandboxGet(c *gin.Context) {
	info := authorized.GetInfo(c)
	locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))

	reg := registry.Global()
	var snaps []taitypes.NodeMeta
	if reg != nil {
		snaps = reg.List()
	}

	// Filter nodes by ownership
	var filtered []taitypes.NodeMeta
	for i := range snaps {
		s := &snaps[i]
		if !taitypes.IsPublicNode(s.Mode) && !sandboxNodeOwnedBy(s, info) {
			continue
		}
		if !s.Capabilities.Docker {
			continue
		}
		filtered = append(filtered, *s)
	}

	mgr := getSandboxManager()

	// Build nodes concurrently
	nodes := make([]ComputerNode, len(filtered))
	var wg sync.WaitGroup
	for i, snap := range filtered {
		wg.Add(1)
		go func(idx int, s taitypes.NodeMeta) {
			defer wg.Done()
			kind := "tai-link"
			switch s.Mode {
			case "local":
				kind = "local"
			case "cloud":
				kind = "cloud"
			}
			node := ComputerNode{
				NodeID:      s.TaiID,
				DisplayName: s.DisplayName,
				Kind:        kind,
				OS:          friendlyOS(s.System.OS),
				Arch:        s.System.Arch,
				CPU:         s.System.NumCPU,
				MemoryGB:    int(s.System.TotalMem / (1024 * 1024 * 1024)),
				Online:      s.Status == "online",
			}
			if node.DisplayName == "" {
				node.DisplayName = s.System.Hostname
			}
			if node.DisplayName == "" {
				node.DisplayName = s.TaiID
			}

			// Fetch Docker info for online nodes
			if node.Online {
				if di := fetchDockerInfo(s.TaiID); di != nil {
					node.DockerVersion = di.Version
					if node.MemoryGB == 0 && di.MemTotal > 0 {
						node.MemoryGB = int(di.MemTotal / (1024 * 1024 * 1024))
					}
					if node.CPU == 0 && di.NCPU > 0 {
						node.CPU = di.NCPU
					}
				}
			}

			// Count running sandboxes
			if mgr != nil {
				boxes, err := mgr.List(context.Background(), sandboxv2.ListOptions{NodeID: s.TaiID})
				if err == nil {
					node.RunningSandboxes = len(boxes)
				}
			}

			nodes[idx] = node
		}(i, snap)
	}
	wg.Wait()

	// Registry config
	regConfig := SandboxRegistryConfig{}
	if setting.Global != nil {
		saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, sandboxRegistryNS)
		if v, ok := saved["registry_url"].(string); ok {
			regConfig.RegistryURL = v
		}
		if v, ok := saved["username"].(string); ok {
			regConfig.Username = v
		}
		if v, ok := saved["password"].(string); ok && v != "" {
			regConfig.Password = cloudMaskKey(cloudDecrypt(v))
		}
	}

	// Collect assistant images (locale-resolved names)
	assistantImages := collectAssistantImages(locale)

	// Build image list per node concurrently
	images := make(map[string][]SandboxImage)
	var imgWg sync.WaitGroup
	var imgMu sync.Mutex
	for _, node := range nodes {
		if !node.Online {
			imgMu.Lock()
			images[node.NodeID] = []SandboxImage{}
			imgMu.Unlock()
			continue
		}
		imgWg.Add(1)
		go func(nodeID string) {
			defer imgWg.Done()
			nodeImages := buildNodeImages(nodeID, assistantImages, locale)
			imgMu.Lock()
			images[nodeID] = nodeImages
			imgMu.Unlock()
		}(node.NodeID)
	}
	imgWg.Wait()

	data := SandboxPageData{
		Nodes:    nodes,
		Registry: regConfig,
		Images:   images,
	}
	if data.Nodes == nil {
		data.Nodes = []ComputerNode{}
	}

	response.RespondWithSuccess(c, http.StatusOK, data)
}

func buildNodeImages(nodeID string, assistantImages map[string][]string, locale string) []SandboxImage {
	res, ok := tai.GetResources(nodeID)
	if !ok || res.Image == nil {
		return []SandboxImage{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	localImages, err := res.Image.List(ctx)
	if err != nil {
		return []SandboxImage{}
	}

	// Build tag index from local images
	tagIndex := make(map[string]runtime.ImageInfo)
	for _, img := range localImages {
		for _, tag := range img.Tags {
			tagIndex[tag] = img
		}
	}

	var result []SandboxImage
	for imageRef, names := range assistantImages {
		imgName, tag := splitImageRef(imageRef)
		si := SandboxImage{
			ID:             imageRefToID(imageRef),
			AssistantNames: names,
			ImageName:      imgName,
			Tag:            tag,
			Status:         "not_downloaded",
		}

		// Check if already downloaded
		if info, ok := tagIndex[imageRef]; ok {
			si.Status = "downloaded"
			si.SizeMB = int(info.Size / (1024 * 1024))
		}

		trackerKey := nodeID + ":" + imageRef
		if v, ok := pullTracker.Load(trackerKey); ok {
			ps := v.(*pullState)
			if !ps.Done {
				si.Status = "downloading"
				p := ps.Progress
				si.Progress = &p
			} else if ps.Error != "" {
				si.Status = "error"
				si.ErrorMessage = friendlyImageError(locale, ps.Error)
			} else {
				si.Status = "downloaded"
			}
		}

		result = append(result, si)
	}

	if result == nil {
		return []SandboxImage{}
	}
	return result
}

// ---------------------------------------------------------------------------
// PUT /setting/sandbox/registry
// ---------------------------------------------------------------------------

func handleSandboxRegistry(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := cloudScope(info)

	var body SandboxRegistryConfig
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	m := map[string]interface{}{
		"registry_url": body.RegistryURL,
		"username":     body.Username,
	}
	if body.Password != "" {
		m["password"] = cloudEncrypt(body.Password)
	} else {
		existing, _ := setting.Global.Get(scope, sandboxRegistryNS)
		if v, ok := existing["password"].(string); ok {
			m["password"] = v
		}
	}

	if _, err := setting.Global.Set(scope, sandboxRegistryNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := SandboxRegistryConfig{
		RegistryURL: body.RegistryURL,
		Username:    body.Username,
	}
	if v, ok := m["password"].(string); ok && v != "" {
		result.Password = cloudMaskKey(cloudDecrypt(v))
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// POST /setting/sandbox/nodes/:nodeId/images/:imageId/pull
// ---------------------------------------------------------------------------

func handleSandboxPull(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	nodeID := c.Param("nodeId")
	imageID := c.Param("imageId")
	imageRef, err := idToImageRef(imageID)
	if err != nil || imageRef == "" {
		respondError(c, http.StatusBadRequest, "invalid image ID")
		return
	}

	res, status, errMsg := getNodeResources(nodeID)
	if res == nil {
		respondError(c, status, errMsg)
		return
	}

	pullOpts := runtime.PullOptions{}
	info := authorized.GetInfo(c)
	if setting.Global != nil {
		saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, sandboxRegistryNS)
		if regURL, ok := saved["registry_url"].(string); ok && regURL != "" {
			if strings.HasPrefix(imageRef, regURL) || strings.HasPrefix(imageRef, strings.TrimPrefix(regURL, "https://")) {
				user, _ := saved["username"].(string)
				pass, _ := saved["password"].(string)
				if user != "" {
					pullOpts.Auth = &runtime.RegistryAuth{
						Username: user,
						Password: cloudDecrypt(pass),
						Server:   regURL,
					}
				}
			}
		}
	}

	trackerKey := nodeID + ":" + imageRef
	log.Info("[sandbox] pull start: trackerKey=%s imageRef=%s", trackerKey, imageRef)
	pullTracker.Store(trackerKey, &pullState{
		ImageRef: imageRef,
		NodeID:   nodeID,
		Progress: 0,
	})

	ch, pullErr := res.Image.Pull(context.Background(), imageRef, pullOpts)
	if pullErr != nil {
		log.Error("[sandbox] pull initiate failed: %s err=%v", trackerKey, pullErr)
		pullTracker.Delete(trackerKey)
		respondError(c, http.StatusBadGateway, "pull failed: "+pullErr.Error())
		return
	}

	if ch != nil {
		log.Info("[sandbox] pull channel received, starting goroutine: %s", trackerKey)
		go consumePullProgress(trackerKey, ch)
	} else {
		log.Info("[sandbox] pull channel is nil, marking as done: %s", trackerKey)
		pullTracker.Store(trackerKey, &pullState{
			ImageRef: imageRef,
			NodeID:   nodeID,
			Progress: 100,
			Done:     true,
		})
	}

	imgName, tag := splitImageRef(imageRef)
	p := 0
	response.RespondWithSuccess(c, http.StatusOK, SandboxImage{
		ID:        imageRefToID(imageRef),
		ImageName: imgName,
		Tag:       tag,
		Status:    "downloading",
		Progress:  &p,
	})
}

func consumePullProgress(trackerKey string, ch <-chan runtime.PullProgress) {
	log.Info("[sandbox] consumePullProgress started: %s", trackerKey)
	var totalBytes int64
	var currentBytes int64
	var eventCount int
	layerProgress := make(map[string]int64)
	layerTotal := make(map[string]int64)

	for p := range ch {
		eventCount++
		if p.Error != "" {
			log.Error("[sandbox] pull error: %s err=%s", trackerKey, p.Error)
			pullTracker.Store(trackerKey, &pullState{
				Done:  true,
				Error: p.Error,
			})
			go func() {
				time.Sleep(60 * time.Second)
				pullTracker.Delete(trackerKey)
			}()
			return
		}

		if p.Layer != "" && p.Total > 0 {
			layerTotal[p.Layer] = p.Total
			layerProgress[p.Layer] = p.Current
		}

		totalBytes = 0
		currentBytes = 0
		for layer, t := range layerTotal {
			totalBytes += t
			currentBytes += layerProgress[layer]
		}

		pct := 0
		if totalBytes > 0 {
			pct = int(currentBytes * 100 / totalBytes)
			if pct > 99 {
				pct = 99
			}
		}

		pullTracker.Store(trackerKey, &pullState{
			Progress: pct,
		})
	}

	log.Info("[sandbox] pull complete (channel closed): %s events=%d", trackerKey, eventCount)
	pullTracker.Store(trackerKey, &pullState{
		Progress: 100,
		Done:     true,
	})
	go func() {
		time.Sleep(60 * time.Second)
		pullTracker.Delete(trackerKey)
	}()
}

// ---------------------------------------------------------------------------
// POST /setting/sandbox/nodes/:nodeId/images/pull-all
// ---------------------------------------------------------------------------

func handleSandboxPullAll(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	nodeID := c.Param("nodeId")
	locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))

	res, status, errMsg := getNodeResources(nodeID)
	if res == nil {
		respondError(c, status, errMsg)
		return
	}

	assistantImages := collectAssistantImages(locale)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	localImages, _ := res.Image.List(ctx)
	tagIndex := make(map[string]bool)
	for _, img := range localImages {
		for _, tag := range img.Tags {
			tagIndex[tag] = true
		}
	}

	// Build pull options
	pullOpts := runtime.PullOptions{}
	info := authorized.GetInfo(c)
	if setting.Global != nil {
		saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, sandboxRegistryNS)
		if regURL, ok := saved["registry_url"].(string); ok && regURL != "" {
			user, _ := saved["username"].(string)
			pass, _ := saved["password"].(string)
			if user != "" {
				pullOpts.Auth = &runtime.RegistryAuth{
					Username: user,
					Password: cloudDecrypt(pass),
					Server:   regURL,
				}
			}
		}
	}

	var result []SandboxImage
	for imageRef, names := range assistantImages {
		if tagIndex[imageRef] {
			continue
		}

		trackerKey := nodeID + ":" + imageRef
		// Skip if already pulling
		if v, ok := pullTracker.Load(trackerKey); ok {
			ps := v.(*pullState)
			if !ps.Done {
				imgName, tag := splitImageRef(imageRef)
				p := ps.Progress
				result = append(result, SandboxImage{
					ID:             imageRefToID(imageRef),
					AssistantNames: names,
					ImageName:      imgName,
					Tag:            tag,
					Status:         "downloading",
					Progress:       &p,
				})
				continue
			}
		}

		pullTracker.Store(trackerKey, &pullState{
			ImageRef: imageRef,
			NodeID:   nodeID,
			Progress: 0,
		})

		ch, pullErr := res.Image.Pull(context.Background(), imageRef, pullOpts)
		if pullErr != nil {
			pullTracker.Delete(trackerKey)
			continue
		}
		if ch != nil {
			go consumePullProgress(trackerKey, ch)
		}

		imgName, tag := splitImageRef(imageRef)
		p := 0
		result = append(result, SandboxImage{
			ID:             imageRefToID(imageRef),
			AssistantNames: names,
			ImageName:      imgName,
			Tag:            tag,
			Status:         "downloading",
			Progress:       &p,
		})
	}

	if result == nil {
		result = []SandboxImage{}
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// DELETE /setting/sandbox/nodes/:nodeId/images/:imageId
// ---------------------------------------------------------------------------

func handleSandboxImageDelete(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	nodeID := c.Param("nodeId")
	imageID := c.Param("imageId")
	imageRef, err := idToImageRef(imageID)
	if err != nil || imageRef == "" {
		respondError(c, http.StatusBadRequest, "invalid image ID")
		return
	}

	res, status, errMsg := getNodeResources(nodeID)
	if res == nil {
		respondError(c, status, errMsg)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if res.Runtime != nil {
		containers, _ := res.Runtime.List(ctx, runtime.ListOptions{All: true})
		for _, ctr := range containers {
			if ctr.Image == imageRef {
				_ = res.Runtime.Remove(ctx, ctr.ID, true)
			}
		}
	}

	if err := res.Image.Remove(ctx, imageRef, true); err != nil {
		locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))
		respondError(c, http.StatusBadRequest, friendlyImageError(locale, err.Error()))
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// POST /setting/sandbox/nodes/:nodeId/check-docker
// ---------------------------------------------------------------------------

func handleSandboxCheckDocker(c *gin.Context) {
	nodeID := c.Param("nodeId")

	reg := registry.Global()
	if reg == nil {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": nil, "message": "tai registry not initialized"})
		return
	}

	meta, ok := reg.Get(nodeID)
	if !ok {
		respondError(c, http.StatusNotFound, "node not found: "+nodeID)
		return
	}

	if meta.Status != "online" {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": nil, "message": "node is offline"})
		return
	}

	res, ok := tai.GetResources(nodeID)
	if !ok || res.Runtime == nil {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": nil, "message": "Docker not available"})
		return
	}

	cli := runtime.DockerCli(res.Runtime)
	if cli == nil {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": nil, "message": "Docker not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ver, err := cli.ServerVersion(ctx)
	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": nil, "message": "Docker check failed: " + err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"docker_version": ver.Version})
}
