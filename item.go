package main

import (
	"github.com/lafriks/go-tiled"
	"image"
)

type item struct {
	picture          image.Image
	displayName      string
	xLoc             int
	yLoc             int
	yAnimationOffset int
	delay            int
	level            *tiled.Map
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

func (item *item) itemAnimate() {
	item.delay++
	if item.delay%6 == 0 {
		item.yAnimationOffset++
		if item.yAnimationOffset > 5 {
			item.yAnimationOffset = 0
		}
	}
}
