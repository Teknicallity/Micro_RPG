package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"path"
	"strings"
)

type worldinfo struct {
	levelCurrent          *tiled.Map
	levelMaps             []*tiled.Map
	tileHashCurrent       map[uint32]*ebiten.Image
	tileHashes            []map[uint32]*ebiten.Image
	pathFindingMapCurrent []string
	pathFindingMaps       [][]string
	pathGridCurrent       *paths.Grid
	pathGrids             []*paths.Grid
}

func (w worldinfo) initializeWorldInfo() worldinfo {
	tileMapHashes := make([]map[uint32]*ebiten.Image, 0, 5)
	levelmaps := make([]*tiled.Map, 0, 5)
	pathfindingmaps := make([][]string, 0, 5)
	pathfindinggrids := make([]*paths.Grid, 0, 5)
	world := worldinfo{
		levelCurrent:          nil,
		levelMaps:             levelmaps,
		tileHashCurrent:       nil,
		tileHashes:            tileMapHashes,
		pathFindingMapCurrent: nil,
		pathFindingMaps:       pathfindingmaps,
		pathGridCurrent:       nil,
		pathGrids:             pathfindinggrids,
	}

	w.importTmx("dirt.tmx")
	w.importTmx("island.tmx")
	w.importTmx("world.tmx")
	return world
}

func (w worldinfo) fillSearchPaths() {

}

func (w worldinfo) importTmx(filename string) {
	gameMap := loadMapFromEmbedded(path.Join("assets", filename))
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)

	w.levelMaps = append(w.levelMaps, gameMap)
	w.levelCurrent = gameMap
	w.tileHashCurrent = ebitenImageMap
	w.tileHashes = append(w.tileHashes, ebitenImageMap)

	searchMap := w.makeSearchMap(gameMap)
	w.pathFindingMapCurrent = searchMap
	w.pathFindingMaps = append(w.pathFindingMaps, searchMap)

	searchablePathMap := paths.NewGridFromStringArrays(searchMap, gameMap.TileWidth, gameMap.TileHeight)
	w.pathGridCurrent = searchablePathMap
	w.pathGrids = append(w.pathGrids, searchablePathMap)

}

// makeSearchMap Takes a tiled.Map and returns a string array, which is used by the paths package
func (w worldinfo) makeSearchMap(tiledMap *tiled.Map) []string {
	mapAsStringSlice := make([]string, 0, tiledMap.Height) //each row will be its own string
	row := strings.Builder{}
	for position, tile := range tiledMap.Layers[0].Tiles {
		if position%tiledMap.Width == 0 && position > 0 { // we get the 2d array as an unrolled one-d array
			mapAsStringSlice = append(mapAsStringSlice, row.String())
			row = strings.Builder{}
		}
		row.WriteString(fmt.Sprintf("%d", tile.ID))
	}
	mapAsStringSlice = append(mapAsStringSlice, row.String())
	return mapAsStringSlice
}
