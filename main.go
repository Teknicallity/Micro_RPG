package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"log"
	"path"
	"slices"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	FRAMES_PER_ROW = 4
	FRAME_COUNT    = 8
	resizeScale    = 3
	worldScale     = 3
)

const (
	DOWN = iota
	RIGHT
	UP
	LEFT
)

const (
	WALK = iota
	ATTACK
)

const (
	CHARACTRIGHT = iota
	CHARACTLEFT
)

type rpgGame struct {
	levelCurrent    *tiled.Map
	levelMaps       []*tiled.Map
	tileHashCurrent map[uint32]*ebiten.Image
	tileHashes      []map[uint32]*ebiten.Image
	windowWidth     int
	windowHeight    int
	barrierIDs      []uint32
	barrierRect     []image.Rectangle
	teleporterRects map[uint32]image.Rectangle
	player          character
	chs             []character
}

type character struct {
	spriteSheet  *ebiten.Image
	xLoc         int
	yLoc         int
	hitPoints    int
	inventory    []item
	direction    int
	frame        int
	frameDelay   int
	FRAME_HEIGHT int
	FRAME_WIDTH  int
	action       int
	imageYOffset int
	speed        int
	level        *tiled.Map
}

type item struct {
	picture     *ebiten.Image
	displayName string
}

func (game *rpgGame) Update() error {
	getPlayerInput(game)

	animatePlayerSprite(&game.player)
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		game.player.action = ATTACK
	} else {
		game.player.action = WALK
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) && game.player.action == WALK {
		game.movePlayer(&game.player.xLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyD) && game.player.action == WALK {
		game.movePlayer(&game.player.xLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyW) && game.player.action == WALK {
		game.movePlayer(&game.player.yLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyS) && game.player.action == WALK {
		game.movePlayer(&game.player.yLoc)
	}
	game.outOfBoundsCheck()
	//fmt.Printf("x: %d, y: %d\n", game.player.xLoc, game.player.yLoc)

	for i := range game.chs {
		if game.chs[i].action == WALK {
			game.chs[i].frameDelay += 1
			if game.chs[i].frameDelay%8 == 0 {
				game.chs[i].frame += 1
				if game.chs[i].frame >= 4 {
					game.chs[i].frame = 0
				}
			}
		}
	}

	//every broder tile, add rectangle to slice, loop through and check collisons

	return nil
}

func (game *rpgGame) movePlayer(location *int) {
	if isBorderColliding(game.barrierRect, &game.player) {
		*location -= translateDirection(game.player.direction) * game.player.speed * 5
	} else {
		*location += translateDirection(game.player.direction) * game.player.speed
	}
}

func translateDirection(direction int) int {
	switch direction {
	case DOWN:
		return 1
	case RIGHT:
		return 1
	case UP:
		return -1
	case LEFT:
		return -1
	default:
		return 0
	}
}

func (game *rpgGame) outOfBoundsCheck() {
	if game.player.xLoc < -100 || game.player.xLoc > game.levelCurrent.TileWidth*game.levelCurrent.Width*worldScale {
		game.player.xLoc = 300
		game.player.yLoc = 300
	} else if game.player.yLoc < -100 || game.player.yLoc > game.levelCurrent.TileHeight*game.levelCurrent.Height*worldScale {
		game.player.xLoc = 300
		game.player.yLoc = 300
	}
}

func (game *rpgGame) Draw(screen *ebiten.Image) {
	//screen.Fill(colornames.Blue)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()

	var teleporterIDs = []uint32{1, 2, 3}
	for _, layer := range game.levelCurrent.Layers {
		for tileY := 0; tileY < game.levelCurrent.Height; tileY++ {
			for tileX := 0; tileX < game.levelCurrent.Width; tileX++ {
				op.GeoM.Reset()
				//get on screen position
				tileXPos := float64(game.levelCurrent.TileWidth * tileX)
				tileYPos := float64(game.levelCurrent.TileHeight * tileY)
				op.GeoM.Translate(tileXPos, tileYPos)

				// Get the tile ID from the appropriate LAYER
				tileToDraw := layer.Tiles[tileY*game.levelCurrent.Width+tileX]

				if slices.Contains(game.barrierIDs, tileToDraw.ID) {
					barrierRectangle := image.Rect(int(tileXPos), int(tileYPos),
						int(tileXPos)+game.levelCurrent.TileWidth, int(tileYPos)+game.levelCurrent.TileHeight)
					game.barrierRect = append(game.barrierRect, barrierRectangle)
				}
				if slices.Contains(teleporterIDs, tileToDraw.ID) {
					teleporterRectangle := image.Rect(int(tileXPos), int(tileYPos),
						int(tileXPos)+game.levelCurrent.TileWidth, int(tileYPos)+game.levelCurrent.TileHeight)
					game.teleporterRects[tileToDraw.ID] = teleporterRectangle
				}

				if tileToDraw.ID != 0 {
					// Retrieve the corresponding sub-image from the map
					ebitenTileToDraw, ok := game.tileHashCurrent[tileToDraw.ID]
					if !ok {
						// Handle the case where the tile ID is not found in the map
						fmt.Printf("Tile ID %d not found in tileHashCurrent\n", tileToDraw.ID)
						continue
					}
					op.GeoM.Scale(worldScale, worldScale)
					// Draw the sub-image
					screen.DrawImage(ebitenTileToDraw, op)
				}
			}
		}
	}

	teleID := getTeleporterCollisionID(game.teleporterRects, &game.player)
	if teleID != 0 {
		game.changeWorldMap(teleID)
	}

	drawPlayerFromSpriteSheet(op, screen, game.player)
	for _, charact := range game.chs {
		if charact.level == game.levelCurrent {
			drawImageFromSpriteSheet(op, screen, charact)
		}
	}

	//DrawCenteredText(screen, font.Face("Comic Sans"), "hello", 200, 100)
	img := ebiten.NewImage(300, 100)
	addLabel(img, 20, 30, "Hello Go")
	op.GeoM.Reset()
	screen.DrawImage(img, op)
}

func drawPlayerFromSpriteSheet(op *ebiten.DrawImageOptions, screen *ebiten.Image, targetCharacter character) {
	op.GeoM.Reset()
	op.GeoM.Scale(resizeScale, resizeScale)
	op.GeoM.Translate(float64(targetCharacter.xLoc), float64(targetCharacter.yLoc))
	screen.DrawImage(targetCharacter.spriteSheet.SubImage(
		image.Rect(
			targetCharacter.frame*targetCharacter.FRAME_WIDTH,
			targetCharacter.direction*targetCharacter.FRAME_HEIGHT,
			targetCharacter.frame*targetCharacter.FRAME_WIDTH+targetCharacter.FRAME_WIDTH,
			targetCharacter.direction*targetCharacter.FRAME_HEIGHT+targetCharacter.FRAME_HEIGHT)).(*ebiten.Image), op)
}

func drawImageFromSpriteSheet(op *ebiten.DrawImageOptions, screen *ebiten.Image, targetCharacter character) {
	op.GeoM.Reset()
	if targetCharacter.direction == CHARACTLEFT {
		op.GeoM.Scale(resizeScale, resizeScale)
		op.GeoM.Translate(float64(targetCharacter.xLoc), float64(targetCharacter.yLoc))
	} else if targetCharacter.direction == CHARACTRIGHT {
		op.GeoM.Scale(-resizeScale, resizeScale)
		op.GeoM.Translate(
			float64(targetCharacter.xLoc)+(float64(targetCharacter.FRAME_WIDTH)*resizeScale), float64(targetCharacter.yLoc))
	}
	screen.DrawImage(targetCharacter.spriteSheet.SubImage(
		image.Rect(
			targetCharacter.frame*targetCharacter.FRAME_WIDTH,
			targetCharacter.imageYOffset*targetCharacter.FRAME_HEIGHT,
			targetCharacter.frame*targetCharacter.FRAME_WIDTH+targetCharacter.FRAME_WIDTH,
			targetCharacter.FRAME_HEIGHT+targetCharacter.FRAME_HEIGHT*targetCharacter.imageYOffset)).(*ebiten.Image), op)
}

func animatePlayerSprite(targetCharacter *character) {
	if targetCharacter.action == WALK {
		targetCharacter.frameDelay += 1
		if targetCharacter.frameDelay%8 == 0 {
			targetCharacter.frame += 1
			if targetCharacter.frame >= 4 {
				targetCharacter.frame = 0
			}
		}
	} else if targetCharacter.action == ATTACK {
		if 4 <= targetCharacter.frame && targetCharacter.frame <= 7 {
			targetCharacter.frameDelay += 1
			if targetCharacter.frameDelay%8 == 0 {
				targetCharacter.frame--
				if targetCharacter.frame <= 4 {
					targetCharacter.frame = 7
				}
			}
		} else {
			targetCharacter.frame = 7
		}
	}
}

func (game *rpgGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight //by default, just return the current dimensions
}

func main() {
	ebiten.SetWindowTitle("SimpleRPG")

	tileMapHashes := make([]map[uint32]*ebiten.Image, 0, 5)
	levelmaps := make([]*tiled.Map, 0, 5)
	gameMap := loadMapFromEmbedded(path.Join("assets", "dirt.tmx"))
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	levelmaps = append(levelmaps, gameMap)
	tileMapHashes = append(tileMapHashes, ebitenImageMap)
	gameMap = loadMapFromEmbedded(path.Join("assets", "island.tmx"))
	ebitenImageMap = makeEbitenImagesFromMap(*gameMap)
	levelmaps = append(levelmaps, gameMap)
	tileMapHashes = append(tileMapHashes, ebitenImageMap)
	gameMap = loadMapFromEmbedded(path.Join("assets", "world.tmx"))
	ebitenImageMap = makeEbitenImagesFromMap(*gameMap)
	levelmaps = append(levelmaps, gameMap)
	tileMapHashes = append(tileMapHashes, ebitenImageMap)

	windowX := gameMap.TileWidth * gameMap.Width * worldScale
	windowY := gameMap.TileHeight * gameMap.Height * worldScale
	ebiten.SetWindowSize(windowX, windowY)

	playerSpriteSheet := LoadEmbeddedImage("characters", "player.png")
	enemySpriteSheet := LoadEmbeddedImage("characters", "characters.png")

	user := character{
		spriteSheet:  playerSpriteSheet,
		xLoc:         250,
		yLoc:         250,
		direction:    RIGHT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  16,
		imageYOffset: -1,
		speed:        3,
		hitPoints:    3,
	}

	mannequin := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         100,
		yLoc:         100,
		inventory:    nil,
		direction:    CHARACTLEFT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 0,
		level:        levelmaps[1],
		hitPoints:    1,
	}

	king := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         100,
		yLoc:         200,
		inventory:    nil,
		direction:    CHARACTLEFT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 1,
		level:        levelmaps[2],
		hitPoints:    1,
	}

	leprechaun := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         300,
		yLoc:         300,
		inventory:    nil,
		direction:    CHARACTRIGHT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 2,
		level:        levelmaps[2],
		hitPoints:    1,
	}
	enemies := make([]character, 0, 5)
	enemies = append(enemies, mannequin)
	enemies = append(enemies, king)
	enemies = append(enemies, leprechaun)

	teleRectangles := map[uint32]image.Rectangle{}

	var barrierID = []uint32{40, 41, 42, 43, 80, 81, 82, 83}

	game := rpgGame{
		levelCurrent:    gameMap,
		tileHashCurrent: ebitenImageMap,
		levelMaps:       levelmaps,
		tileHashes:      tileMapHashes,
		player:          user,
		chs:             enemies,
		barrierIDs:      barrierID,
		windowWidth:     windowX,
		windowHeight:    windowY,
		teleporterRects: teleRectangles,
	}
	err := ebiten.RunGame(&game)
	if err != nil {
		fmt.Println("Failed to run game", err)
	}
}

func LoadEmbeddedImage(folderName string, imageName string) *ebiten.Image {
	embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		log.Fatal("failed to load embedded image ", imageName, err)
	}
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tile image:", imageName, err)
	}
	return ebitenImage
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name, tiled.WithFileSystem(EmbeddedAssets))
	if err != nil {
		fmt.Println("Error loading embedded map:", err)
	}
	return embeddedMap
}

func makeEbitenImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
	idToImage := make(map[uint32]*ebiten.Image)
	tilesetImagePath := path.Join("assets", tiledMap.Tilesets[0].Image.Source)
	embeddedFile, err := EmbeddedAssets.Open(tilesetImagePath)
	if err != nil {
		log.Fatal("failed to load embedded image ", tilesetImagePath, err)
	}
	ebitenImageTileset, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tileset image:", tilesetImagePath, err)
	}
	for _, layer := range tiledMap.Layers {
		for _, tile := range layer.Tiles {

			if _, ok := idToImage[tile.ID]; !ok { //if tileID does not exists
				x := int((tile.ID)%uint32(tiledMap.Tilesets[0].Columns)) * tiledMap.TileWidth
				y := int((tile.ID)/uint32(tiledMap.Tilesets[0].Columns)) * tiledMap.TileHeight
				subImageRect := image.Rect(x, y, x+tiledMap.TileWidth, y+tiledMap.TileHeight)
				subImage := ebitenImageTileset.SubImage(subImageRect).(*ebiten.Image)
				idToImage[tile.ID] = subImage
			} else {
				//do nothing?
			}
		}
	}

	return idToImage
}

func getPlayerInput(game *rpgGame) {
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		game.player.direction = LEFT
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		game.player.direction = RIGHT
	} else if ebiten.IsKeyPressed(ebiten.KeyW) {
		game.player.direction = UP
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		game.player.direction = DOWN
	}
}

func DrawCenteredText(screen *ebiten.Image, font font.Face, s string, cx, cy int) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(font, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2
	text.Draw(screen, s, font, x, y, colornames.White)
}

func addLabel(img *ebiten.Image, x, y int, label string) {
	// from https://stackoverflow.com/a/38300583
	col := color.RGBA{200, 100, 0, 255}
	point := fixed.Point26_6{fixed.I(x), fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func isBorderColliding(borderRects []image.Rectangle, player *character) bool {
	playerBounds := collision.BoundingBox{
		X:      float64(player.xLoc),
		Y:      float64(player.yLoc),
		Width:  float64(player.FRAME_WIDTH * resizeScale),
		Height: float64(player.FRAME_HEIGHT * resizeScale),
	}

	for _, border := range borderRects {
		borderBounds := collision.BoundingBox{
			X:      float64(border.Min.X * worldScale),
			Y:      float64(border.Min.Y * worldScale),
			Width:  float64(border.Dx() * worldScale),
			Height: float64(border.Dy() * worldScale),
		}
		if collision.AABBCollision(playerBounds, borderBounds) {
			return true
		}
	}
	return false
}

func getTeleporterCollisionID(teleporterRects map[uint32]image.Rectangle, player *character) uint32 {
	playerBounds := collision.BoundingBox{
		X:      float64(player.xLoc),
		Y:      float64(player.yLoc),
		Width:  float64(player.FRAME_WIDTH * resizeScale),
		Height: float64(player.FRAME_HEIGHT * resizeScale),
	}
	for ID, teleporter := range teleporterRects {
		teleporterBounds := collision.BoundingBox{
			X:      float64(teleporter.Min.X * worldScale),
			Y:      float64(teleporter.Min.Y * worldScale),
			Width:  float64(teleporter.Dx() * worldScale),
			Height: float64(teleporter.Dy() * worldScale),
		}
		if collision.AABBCollision(playerBounds, teleporterBounds) {
			return ID
		}
	}
	return 0
}

//if tileID ==1 change to x map  playerpositionX-screen absolute value

func (game *rpgGame) changeWorldMap(tileID uint32) {
	//
	if tileID == 1 {
		// go to right world
		game.levelCurrent = game.levelMaps[1]
		game.tileHashCurrent = game.tileHashes[1]
		game.player.xLoc = 50
	} else if tileID == 2 {
		// go to main world
		game.levelCurrent = game.levelMaps[2]
		game.tileHashCurrent = game.tileHashes[2]
		if game.player.xLoc > 600 {
			game.player.xLoc = 50
		} else if game.player.xLoc < 150 {
			game.player.xLoc = game.windowWidth - 100
		}
	} else if tileID == 3 {
		//go to left world
		game.levelCurrent = game.levelMaps[0]
		game.tileHashCurrent = game.tileHashes[0]
		game.player.xLoc = game.windowWidth - 100
	}
	//fmt.Println(game.levelCurrent)
	game.barrierRect = game.barrierRect[:0]
	game.teleporterRects = make(map[uint32]image.Rectangle)
}

func (character *character) death() {

}

func itemPickup() {

}
