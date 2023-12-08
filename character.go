package main

import (
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"image"
)

type character struct {
	spriteSheet        *ebiten.Image
	xLoc               int
	yLoc               int
	hitPoints          int
	inventory          []item
	direction          int
	frame              int
	frameDelay         int
	FRAME_HEIGHT       int
	FRAME_WIDTH        int
	action             int
	imageYOffset       int
	speed              int
	level              *tiled.Map
	interactRect       image.Rectangle
	interactCooldown   int
	attackPower        int
	path               *paths.Path
	pathUpdateCooldown int
}

// character method
func (character *character) isPlayerInAttackRange(player *player) bool {
	player.updatePlayerInteractionRectangle()
	npcBounds := character.getCollisionBoundingBox()
	playerBounds := player.getCollisionBoundingBox()

	if collision.AABBCollision(playerBounds, npcBounds) {
		return true
	}
	return false
}

// character
func (character *character) death(game *rpgGame) {
	character.dropAllItems(game)
	character.xLoc = -100
	character.yLoc = -100
	game.sounds.enemyDeath.playSound()
}

// character method
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

// character method
func (character *character) getCollisionBoundingBox() collision.BoundingBox {
	boundBox := collision.BoundingBox{
		X:      float64(character.xLoc),
		Y:      float64(character.yLoc),
		Width:  float64(character.FRAME_WIDTH * resizeScale),
		Height: float64(character.FRAME_HEIGHT * resizeScale),
	}
	return boundBox
}

// character
func (character *character) dropAllItems(game *rpgGame) {
	for i := range character.inventory {
		character.dropItem(game, i)
	}
	character.dropItem(game, -1)
}

// character
func (character *character) dropItem(game *rpgGame, itemIndex int) {
	//character.inventory[itemIndex] = nil
	if itemIndex < 0 {
		heart := HeartItem
		heart.xLoc = character.xLoc + 20
		heart.yLoc = character.yLoc + 20
		heart.level = character.level
		game.droppedItems = append(game.droppedItems, heart)
	} else {
		droppedItem := character.inventory[itemIndex]
		droppedItem.xLoc = character.xLoc + 40
		droppedItem.yLoc = character.yLoc + 40
		droppedItem.level = character.level
		game.droppedItems = append(game.droppedItems, droppedItem)
		character.removeInventoryItemAtIndex(itemIndex)
	}
}

// player

// character
func (character *character) removeInventoryItemAtIndex(index int) {
	retained := make([]item, 0)
	for i := range character.inventory {
		if i != index {
			retained = append(retained, character.inventory[i])
		}
	}
	character.inventory = retained
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

func (character *character) moveCharacter(x, y int, game *rpgGame) {

}
