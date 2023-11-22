package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"log"
	"path"
	"slices"
	"strconv"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	resizeScale     = 3
	worldScale      = 3
	COOLDOWN        = 60
	soundSampleRate = 48000
)

const (
	DOWN = iota
	RIGHT
	UP
	LEFT
)

const (
	WALK = iota
	INTERACT
)

const (
	CHARACTRIGHT = iota
	CHARACTLEFT
)

const (
	NOTTALKED = iota
	TALKED
	RETURNEDITEM
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
	enemies         []character
	questGiver      character
	fontLarge       font.Face
	fontSmall       font.Face
	heartImage      image.Image
	droppedItems    []item
	sounds          sounds
}

type character struct {
	spriteSheet      *ebiten.Image
	xLoc             int
	yLoc             int
	hitPoints        int
	inventory        []item
	direction        int
	frame            int
	frameDelay       int
	FRAME_HEIGHT     int
	FRAME_WIDTH      int
	action           int
	imageYOffset     int
	speed            int
	level            *tiled.Map
	interactRect     image.Rectangle
	interactCooldown int
	questProgress    int
	attackPower      int
}

type item struct {
	picture          image.Image
	displayName      string
	xLoc             int
	yLoc             int
	yAnimationOffset int
	delay            int
}

type sounds struct {
	enemyDeath     sound
	enemyHit       sound
	attackPowerUp  sound
	heal           sound
	playerInteract sound
	playerDamaged  sound
	questGiverTalk sound
	itemPickup     sound

	audioContext *audio.Context
}

type sound struct {
	audioPlayer *audio.Player
}

func (sound *sound) playSound() {
	//sound.
	err := sound.audioPlayer.Rewind()
	if err != nil {
		fmt.Println("Error rewinding sound: ", err)
	}
	sound.audioPlayer.Play()
}

func loadEmbeddedWavToSound(name string, context *audio.Context) sound {
	file, err := EmbeddedAssets.Open(path.Join("assets", "sounds", name))
	if err != nil {
		fmt.Println("Error Loading embedded sound: ", err)
	}
	soundWav, err := wav.DecodeWithoutResampling(file)
	if err != nil {
		fmt.Println("Error interpreting sound file: ", err)
	}
	soundPlay, err := context.NewPlayer(soundWav)
	if err != nil {
		fmt.Println("Couldn't create sound player: ", err)
	}
	return sound{soundPlay}
}

var HeartItem = item{
	picture:          grabItemImage(63, 0, 16, 16),
	displayName:      "Heart",
	xLoc:             400,
	yLoc:             100,
	yAnimationOffset: 0,
	delay:            0,
}

var BookItem = item{
	picture:          grabItemImage(304, 0, 16, 16),
	displayName:      "Book",
	xLoc:             500,
	yLoc:             500,
	yAnimationOffset: 0,
	delay:            0,
}

var StoneItem = item{
	picture:          grabItemImage(256, 16, 16, 16),
	displayName:      "Stone",
	xLoc:             200,
	yLoc:             500,
	yAnimationOffset: 0,
	delay:            0,
}

func (game *rpgGame) Update() error {
	getPlayerInput(game)

	animatePlayerSprite(&game.player)
	game.animateDroppedItems()

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		game.player.action = INTERACT
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
	game.itemsPickupCheck()
	if game.player.convertHeartItemsToHealth() {
		game.sounds.heal.playSound()
	}

	if game.player.action == INTERACT && game.player.interactCooldown < 0 {
		game.sounds.playerInteract.playSound()
		game.player.interactCooldown = COOLDOWN
		for i := range game.enemies {
			if game.enemies[i].level == game.levelCurrent {
				if game.player.playerInteractWithCheck(&game.enemies[i]) {
					game.enemies[i].hitPoints -= game.player.attackPower
					game.sounds.enemyHit.playSound()
					if game.enemies[i].hitPoints == 0 {
						game.enemies[i].death(game)
					}
				}
			}
		}
		if game.player.playerInteractWithCheck(&game.questGiver) && game.questGiver.level == game.levelCurrent {
			if game.player.questProgress == NOTTALKED {
				game.player.questProgress = TALKED
				//display quest text
				game.sounds.questGiverTalk.playSound()
			} else if game.player.questProgress == TALKED && game.player.questCheckForBook() { //AND H IASTEM
				game.player.questProgress = RETURNEDITEM
				game.player.attackPower++
				game.sounds.attackPowerUp.playSound()
			} else if game.player.questProgress == RETURNEDITEM {
				//display another thank you message?
			}
		}
	} else if game.player.interactCooldown > -10 {
		game.player.interactCooldown--
	}
	game.enemiesAttack()

	for i := range game.enemies {
		game.enemies[i].animateCharacter()
	}
	game.questGiver.animateCharacter()

	return nil
}

func (character *character) animateCharacter() {
	if character.action == WALK {
		character.frameDelay += 1
		if character.frameDelay%8 == 0 {
			character.frame += 1
			if character.frame >= 4 {
				character.frame = 0
			}
		}
	}
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
	for _, charact := range game.enemies {
		if charact.level == game.levelCurrent {
			drawImageFromSpriteSheet(op, screen, charact)
		}
	}
	if game.questGiver.level == game.levelCurrent {
		drawImageFromSpriteSheet(op, screen, game.questGiver)
	}

	for _, item := range game.droppedItems {
		op.GeoM.Reset()
		op.GeoM.Scale(resizeScale-1, resizeScale-1)
		op.GeoM.Translate(float64(item.xLoc), float64(item.yLoc-item.yAnimationOffset))
		screen.DrawImage(item.picture.(*ebiten.Image), op)
	}

	game.drawPlayerHealth(op, screen)
	if game.questGiver.level == game.levelCurrent {
		switch game.player.questProgress {
		case TALKED:
			DrawCenteredText(screen, game.fontSmall, "My brother stole my book,\n   please get it back!",
				game.questGiver.xLoc+45, game.questGiver.yLoc)

		case RETURNEDITEM:
			DrawCenteredText(screen, game.fontSmall, "  Thank You!\nI've blessed you\n  with strength",
				game.questGiver.xLoc+45, game.questGiver.yLoc)
		}
	}

	DrawCenteredText(screen, game.fontSmall, "Power:", 50, 700)
	DrawCenteredText(screen, game.fontSmall, strconv.Itoa(game.player.attackPower), 120, 700)

	if game.player.hitPoints <= 0 {
		DrawCenteredText(screen, game.fontLarge, "GAME OVER", game.windowHeight/2, game.windowWidth/2)
	}
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

func (game *rpgGame) drawPlayerHealth(op *ebiten.DrawImageOptions, screen *ebiten.Image) {
	op.GeoM.Reset()
	op.GeoM.Scale(worldScale, worldScale)
	for i := 0; i < game.player.hitPoints; i++ {
		screen.DrawImage(game.heartImage.(*ebiten.Image), op)
		op.GeoM.Translate(16*worldScale, 0)
	}
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
	} else if targetCharacter.action == INTERACT {
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

func (game *rpgGame) animateDroppedItems() {
	for i := range game.droppedItems {
		game.droppedItems[i].itemAnimate()
	}
}

func (item *item) itemAnimate() {
	item.delay++
	if item.delay%6 == 0 {
		item.yAnimationOffset++
		if item.yAnimationOffset > 5 {
			item.yAnimationOffset = 0
		}
	}
}

func (game *rpgGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight //by default, just return the current dimensions
}

func main() {
	ebiten.SetWindowTitle("SimpleRPG")

	soundContext := audio.NewContext(soundSampleRate)

	snds := sounds{
		enemyDeath:     loadEmbeddedWavToSound("enemyDeath.wav", soundContext),
		enemyHit:       loadEmbeddedWavToSound("enemyHit.wav", soundContext),
		attackPowerUp:  loadEmbeddedWavToSound("attackPowerUp.wav", soundContext),
		heal:           loadEmbeddedWavToSound("heal.wav", soundContext),
		playerInteract: loadEmbeddedWavToSound("playerInteract.wav", soundContext),
		playerDamaged:  loadEmbeddedWavToSound("playerDamaged.wav", soundContext),
		questGiverTalk: loadEmbeddedWavToSound("questGiverTalk.wav", soundContext),
		itemPickup:     loadEmbeddedWavToSound("itemPickup.wav", soundContext),
	}

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
	fmt.Printf("windowWidth: %d, windowHeight: %d\n", windowX, windowY)

	playerSpriteSheet := LoadEmbeddedImage("characters", "player.png")
	enemySpriteSheet := LoadEmbeddedImage("characters", "characters.png")

	user := character{
		spriteSheet:      playerSpriteSheet,
		xLoc:             400,
		yLoc:             400,
		direction:        RIGHT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      16,
		imageYOffset:     -1,
		speed:            3,
		hitPoints:        3,
		questProgress:    NOTTALKED,
		interactCooldown: COOLDOWN / 2,
		attackPower:      1,
	}
	questGiver := character{
		spriteSheet:      enemySpriteSheet,
		xLoc:             200,
		yLoc:             150,
		inventory:        nil,
		direction:        CHARACTLEFT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      32,
		action:           WALK,
		imageYOffset:     0,
		level:            levelmaps[2],
		hitPoints:        1,
		interactCooldown: COOLDOWN,
	}

	mannequinInventory := make([]item, 0)
	mannequinInventory = append(mannequinInventory, BookItem)

	mannequin := character{
		spriteSheet:      enemySpriteSheet,
		xLoc:             100,
		yLoc:             100,
		inventory:        mannequinInventory,
		direction:        CHARACTLEFT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      32,
		action:           WALK,
		imageYOffset:     0,
		level:            levelmaps[1],
		hitPoints:        2,
		interactCooldown: COOLDOWN,
		attackPower:      1,
	}

	king := character{
		spriteSheet:      enemySpriteSheet,
		xLoc:             100,
		yLoc:             200,
		inventory:        nil,
		direction:        CHARACTLEFT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      32,
		action:           WALK,
		imageYOffset:     1,
		level:            levelmaps[0],
		hitPoints:        2,
		interactCooldown: COOLDOWN,
		attackPower:      1,
	}

	leprechaun := character{
		spriteSheet:      enemySpriteSheet,
		xLoc:             300,
		yLoc:             300,
		inventory:        nil,
		direction:        CHARACTRIGHT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      32,
		action:           WALK,
		imageYOffset:     2,
		level:            levelmaps[0],
		hitPoints:        2,
		interactCooldown: COOLDOWN,
		attackPower:      1,
	}
	enemies := make([]character, 0, 5)
	enemies = append(enemies, mannequin)
	enemies = append(enemies, king)
	enemies = append(enemies, leprechaun)

	heartImage := grabItemImage(63, 0, 16, 16)
	droppedItems := make([]item, 0, 10)
	heart := HeartItem
	droppedItems = append(droppedItems, heart)
	stone := StoneItem
	droppedItems = append(droppedItems, stone)
	fmt.Printf("items: %d\n", droppedItems)

	teleRectangles := map[uint32]image.Rectangle{}

	var barrierID = []uint32{40, 41, 42, 43, 80, 81, 82, 83}

	game := rpgGame{
		levelCurrent:    gameMap,
		tileHashCurrent: ebitenImageMap,
		levelMaps:       levelmaps,
		tileHashes:      tileMapHashes,
		player:          user,
		enemies:         enemies,
		barrierIDs:      barrierID,
		windowWidth:     windowX,
		windowHeight:    windowY,
		teleporterRects: teleRectangles,
		heartImage:      heartImage,
		fontLarge:       LoadScoreFont(60),
		fontSmall:       LoadScoreFont(16),
		droppedItems:    droppedItems,
		questGiver:      questGiver,
		sounds:          snds,
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

func grabItemImage(startX, startY, width, height int) image.Image {
	spriteSheet := LoadEmbeddedImage("", "objects.png")
	subImageRect := image.Rect(startX, startY, startX+width, startY+height)

	subImage := ebiten.NewImageFromImage(spriteSheet).SubImage(subImageRect)
	return subImage
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

func LoadScoreFont(size float64) font.Face {
	//originally inspired by https://www.fatoldyeti.com/posts/roguelike16/
	trueTypeFont, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		fmt.Println("Error loading font for score:", err)
	}
	fontFace, err := opentype.NewFace(trueTypeFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Println("Error loading font of correct size for score:", err)
	}
	return fontFace
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
	//usage
	//img := ebiten.NewImage(300, 100)
	//addLabel(img, 20, 30, "Hello Go")
	//op.GeoM.Reset()
	//screen.DrawImage(img, op)
}

func isBorderColliding(borderRects []image.Rectangle, player *character) bool {
	playerBounds := player.getCollisionBoundingBox()

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
	playerBounds := player.getCollisionBoundingBox()

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

func (player *character) playerInteractWithCheck(target *character) bool {
	player.updatePlayerInteractRectangle()
	fmt.Printf("%d", player.direction)
	fmt.Printf("player X: %d, Y: %d\n", player.xLoc, player.yLoc)
	fmt.Printf("itneractbox X: %d, Y: %d  width: %d, height: %d\n", player.interactRect.Min.X, player.interactRect.Min.Y, player.interactRect.Dx()*worldScale, player.interactRect.Dy()*worldScale)
	targetBounds := target.getCollisionBoundingBox()
	playerBounds := player.getCollisionBoundingBox()
	playerInteractBounds := collision.BoundingBox{
		X:      float64(player.interactRect.Min.X),
		Y:      float64(player.interactRect.Min.Y),
		Width:  float64(player.interactRect.Dx()),
		Height: float64(player.interactRect.Dy()),
	}
	if collision.AABBCollision(playerBounds, targetBounds) || collision.AABBCollision(playerInteractBounds, targetBounds) {
		return true
	}
	return false
}

func (player *character) updatePlayerInteractRectangle() {
	//based on direction, change targetRectangle
	switch player.direction {
	case DOWN:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc+player.FRAME_HEIGHT*resizeScale,
			player.xLoc+player.FRAME_WIDTH*resizeScale,
			player.yLoc+player.FRAME_HEIGHT*resizeScale+player.FRAME_WIDTH,
		)
	case RIGHT:
		player.interactRect = image.Rect(
			player.xLoc+player.FRAME_WIDTH*resizeScale,
			player.yLoc,
			player.xLoc+(player.FRAME_WIDTH*resizeScale*2),
			player.yLoc+player.FRAME_HEIGHT,
		)
	case UP:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc,
			player.xLoc+(player.FRAME_WIDTH*resizeScale),
			player.yLoc-(player.FRAME_WIDTH*resizeScale)-player.FRAME_WIDTH,
		)
	case LEFT:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc,
			player.xLoc-player.FRAME_WIDTH*resizeScale,
			player.yLoc+player.FRAME_HEIGHT,
		)
	}
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

func (game *rpgGame) enemiesAttack() {
	for i := range game.enemies {
		if game.enemies[i].npcAttackBoundsCheck(&game.player) && game.enemies[i].interactCooldown < 0 {
			if game.enemies[i].level == game.levelCurrent {
				//damage player
				game.sounds.playerDamaged.playSound()
				game.player.hitPoints -= game.enemies[i].attackPower
				game.enemies[i].interactCooldown = COOLDOWN
			}
		} else if game.enemies[i].interactCooldown > -10 {
			game.enemies[i].interactCooldown--
		}
	}

}

func (npc *character) npcAttackBoundsCheck(player *character) bool {
	player.updatePlayerInteractRectangle()
	npcBounds := npc.getCollisionBoundingBox()
	playerBounds := player.getCollisionBoundingBox()

	if collision.AABBCollision(playerBounds, npcBounds) {
		return true
	}
	return false
}

func (character *character) death(game *rpgGame) {
	character.dropAllItems(game)
	character.xLoc = -100
	character.yLoc = -100
	game.sounds.enemyDeath.playSound()
}

func (game *rpgGame) itemsPickupCheck() {
	newDroppedItems := make([]item, 0)
	for i := range game.droppedItems {
		if game.player.isItemColliding(&game.droppedItems[i]) {
			game.player.inventory = append(game.player.inventory, game.droppedItems[i])
			if game.droppedItems[i].displayName != "Heart" {
				game.sounds.itemPickup.playSound()
			}
		} else {
			newDroppedItems = append(newDroppedItems, game.droppedItems[i])
		}
	}
	game.droppedItems = newDroppedItems
}

func (character *character) isItemColliding(item *item) bool {
	itemBounds := collision.BoundingBox{
		X:      float64(item.xLoc),
		Y:      float64(item.yLoc),
		Width:  float64(item.picture.Bounds().Dx() * resizeScale),
		Height: float64(item.picture.Bounds().Dy() * resizeScale),
	}
	playerBounds := character.getCollisionBoundingBox()

	if collision.AABBCollision(itemBounds, playerBounds) {
		return true
	} else {
		return false
	}
}

func (character *character) getCollisionBoundingBox() collision.BoundingBox {
	boundBox := collision.BoundingBox{
		X:      float64(character.xLoc),
		Y:      float64(character.yLoc),
		Width:  float64(character.FRAME_WIDTH * resizeScale),
		Height: float64(character.FRAME_HEIGHT * resizeScale),
	}
	return boundBox
}

func (character *character) dropAllItems(game *rpgGame) {
	for i := range character.inventory {
		character.dropItem(game, i)
	}
	character.dropItem(game, -1)
}

func (character *character) dropItem(game *rpgGame, itemIndex int) {
	//character.inventory[itemIndex] = nil
	if itemIndex < 0 {
		heart := HeartItem
		heart.xLoc = character.xLoc + 20
		heart.yLoc = character.yLoc + 20
		game.droppedItems = append(game.droppedItems, heart)
	} else {
		droppedItem := character.inventory[itemIndex]
		droppedItem.xLoc = character.xLoc + 40
		droppedItem.yLoc = character.yLoc + 40
		game.droppedItems = append(game.droppedItems, droppedItem)
		character.removeInventoryItemAtIndex(itemIndex)
	}
}

func (character *character) getItemIndex(itemName string) int {
	for i := range character.inventory {
		if character.inventory[i].displayName == itemName {
			return i
		}
	}
	return -1
}

func (character *character) questCheckForBook() bool {
	index := character.getItemIndex(BookItem.displayName)
	if index != -1 {
		character.removeInventoryItemAtIndex(index)
		return true
	}
	return false
}

func (character *character) removeInventoryItemAtIndex(index int) {
	retained := make([]item, 0)
	for i := range character.inventory {
		if i != index {
			retained = append(retained, character.inventory[i])
		}
	}
	character.inventory = retained
}

func playSound(s string) {
	fmt.Println(s)
}

func (character *character) convertHeartItemsToHealth() bool {
	indexToRemove := 0
	remove := false
	for i := range character.inventory {
		if character.inventory[i].displayName == "Heart" {
			indexToRemove = i
			remove = true
			character.hitPoints++
			break
		}
	}
	if remove {
		character.removeInventoryItemAtIndex(indexToRemove)
	}
	return remove
}
