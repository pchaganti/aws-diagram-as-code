// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ctl

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/awslabs/diagram-as-code/internal/cache"
	"github.com/awslabs/diagram-as-code/internal/definition"
	"github.com/awslabs/diagram-as-code/internal/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

func stringToColor(c string) color.RGBA {
	var r, g, b, a uint8
	_, err := fmt.Sscanf(c, "rgba(%d,%d,%d,%d)", &r, &g, &b, &a)
	if err != nil {
		log.Fatal(err)
	}
	return color.RGBA{r, g, b, a}
}

type TemplateStruct struct {
	Diagram `yaml:"Diagram"`
}

type Diagram struct {
	DefinitionFiles []DefinitionFile    `yaml:"DefinitionFiles"`
	Resources       map[string]Resource `yaml:"Resources"`
	Links           []Link              `yaml:"Links"`
}

type DefinitionFile struct {
	Type      string                         `yaml:"Type"` // URL,LocalFile,Embed
	Url       string                         `yaml:"Url"`
	LocalFile string                         `yaml:"LocalFile"`
	Embed     definition.DefinitionStructure `yaml:"Embed"`
}

type Resource struct {
	Type       string   `yaml:"Type"`
	Icon       string   `yaml:"Icon"`
	Direction  string   `yaml:"Direction"`
	Preset     string   `yaml:"Preset"`
	Align      string   `yaml:"Align"`
	FillColor  string   `yaml:"FillColor"`
	Title      string   `yaml:"Title"`
	TitleColor string   `yaml:"TitleColor"`
	Font       string   `yaml:"Font"`
	Children   []string `yaml:"Children"`
}

type Link struct {
	Source          string          `yaml:"Source"`
	SourcePosition  string          `yaml:"SourcePosition"`
	SourceArrowHead types.ArrowHead `yaml:"SourceArrowHead"`
	Target          string          `yaml:"Target"`
	TargetPosition  string          `yaml:"TargetPosition"`
	TargetArrowHead types.ArrowHead `yaml:"TargetArrowHead"`
	Type            string          `yaml:"Type"`
	LineWidth       int             `yaml:"LineWidth"`
}

func createDiagram(resources map[string]*types.Resource, outputfile *string) {

	log.Info("--- Draw diagram ---")
	resources["Canvas"].Scale(nil)
	resources["Canvas"].ZeroAdjust()
	img := resources["Canvas"].Draw(nil, nil)

	log.Infof("Save %s\n", *outputfile)
	fmt.Printf("[Completed] AWS infrastructure diagram generated: %s\n", *outputfile)
	f, _ := os.OpenFile(*outputfile, os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()
	png.Encode(f, img)
}

func loadDefinitionFiles(template *TemplateStruct, ds *definition.DefinitionStructure) {

	// Load definition files
	for _, v := range template.Diagram.DefinitionFiles {
		switch v.Type {
		case "URL":
			log.Infof("Fetch definition file from URL: %s\n", v.Url)
			cacheFilePath, err := cache.FetchFile(v.Url)
			if err != nil {
				log.Fatal(err)
			}
			log.Infof("Read definition file from cache file: %s\n", cacheFilePath)
			err = ds.LoadDefinitions(cacheFilePath)
			if err != nil {
				log.Fatal(err)
			}
		case "LocalFile":
			log.Infof("Read definition file from path: %s\n", v.LocalFile)
			err := ds.LoadDefinitions(v.LocalFile)
			if err != nil {
				log.Fatal(err)
			}
		case "Embed":
			log.Info("Read embedded definitions")
			maps.Copy(ds.Definitions, v.Embed.Definitions)
		}
	}

}

func loadResources(template *TemplateStruct, ds definition.DefinitionStructure, resources map[string]*types.Resource) {

	resources["Canvas"] = new(types.Resource).Init()

	for k, v := range template.Resources {
		title := v.Title

		log.Infof("Load Resource: %s (%s)\n", k, v.Type)
		switch v.Type {
		case "":
			log.Infof("%s does not have Type. Delete it from resources", k)
			delete(resources, k)
		case "AWS::Diagram::Canvas":
			resources[k].SetBorderColor(color.RGBA{0, 0, 0, 0})
			resources[k].SetFillColor(color.RGBA{255, 255, 255, 255})
		case "AWS::Diagram::Resource":
			resources[k] = new(types.Resource).Init()
		case "AWS::Diagram::VerticalStack":
			resources[k] = new(types.VerticalStack).Init()
		case "AWS::Diagram::HorizontalStack":
			resources[k] = new(types.HorizontalStack).Init()
		default:
			def, ok := ds.Definitions[v.Type]
			if !ok {
				newType := fallbackToServiceIcon(v.Type)
				_, check := ds.Definitions[newType]
				if !check {
					log.Warnf("Type %s is not defined in the DAC definition file. It cannot be fall backed to service icon. Ignore this type.\n", v.Type)
					continue
				}
				log.Warnf("Type %s is not defined in the DAC definition file. It's fall backed to its service icon (Type %s).\n", v.Type, newType)
				def = ds.Definitions[newType]

				// Change the title to indicate the original resource type for fallback icons.
				if title == "" {
					v.Title = v.Type
					title = v.Type
				}
			}
			if def.Type == "Resource" {
				resources[k] = new(types.Resource).Init()
			} else if def.Type == "Group" {
				resources[k] = new(types.Resource).Init()
			}
			if fill := def.Fill; fill != nil {
				resources[k].SetFillColor(stringToColor(fill.Color))
			}
			if border := def.Border; border != nil {
				resources[k].SetBorderColor(stringToColor(border.Color))
				switch border.Type {
				case "straight":
					resources[k].SetBorderType(types.BORDER_TYPE_STRAIGHT)
				case "dashed":
					resources[k].SetBorderType(types.BORDER_TYPE_DASHED)
				default:
					resources[k].SetBorderType(types.BORDER_TYPE_STRAIGHT)
				}
			}
			if label := def.Label; label != nil {
				if label.Title != "" {
					resources[k].SetLabel(&label.Title, nil, nil)
				}
				if label.Color != "" {
					c := stringToColor(label.Color)
					resources[k].SetLabel(nil, &c, nil)
				}
				if label.Font != "" {
					resources[k].SetLabel(nil, nil, &label.Font)
				}
			}
			if headerAlign := def.HeaderAlign; headerAlign != "" {
				resources[k].SetHeaderAlign(headerAlign)
			}
			if icon := def.Icon; icon != nil {
				if def.CacheFilePath == "" {
					break
				}
				resources[k].LoadIcon(def.CacheFilePath)
			}
		}

		switch v.Preset {
		case "BlankGroup":
			resources[k].SetIconBounds(image.Rect(0, 0, 64, 64))
			resources[k].SetBorderColor(color.RGBA{0, 0, 0, 0})
		case "":
		default:
			def, ok := ds.Definitions[v.Preset]
			if !ok {
				log.Warnf("Unknown preset: %s\n", v.Type)
				continue
			}
			if fill := def.Fill; fill != nil {
				resources[k].SetFillColor(stringToColor(fill.Color))
			}
			if border := def.Border; border != nil {
				resources[k].SetBorderColor(stringToColor(border.Color))
			}
			if label := def.Label; label != nil {
				if label.Title != "" {
					resources[k].SetLabel(&label.Title, nil, nil)
				}
				if label.Color != "" {
					c := stringToColor(label.Color)
					resources[k].SetLabel(nil, &c, nil)
				}
				if label.Font != "" {
					resources[k].SetLabel(nil, nil, &label.Font)
				}

			}
			if icon := def.Icon; icon != nil {
				if def.CacheFilePath == "" {
					continue
				}
				resources[k].LoadIcon(def.CacheFilePath)
			}
		}
		if v.Icon != "" {
			resources[k].LoadIcon(v.Icon)
		}
		if v.Title != "" {
			resources[k].SetLabel(&title, nil, nil)
		}
		if v.TitleColor != "" {
			c := stringToColor(v.TitleColor)
			resources[k].SetLabel(nil, &c, nil)
		}
		if v.Font != "" {
			resources[k].SetLabel(nil, nil, &v.Font)
		}
		if v.Align != "" {
			resources[k].SetAlign(v.Align)
		}
		if v.Direction != "" {
			resources[k].SetDirection(v.Direction)
		}
		if v.FillColor != "" {
			resources[k].SetFillColor(stringToColor(v.FillColor))
		}
	}

}

func fallbackToServiceIcon(inputType string) string {

	parts := strings.SplitN(inputType, "::", 3)
	possibleServiceType := strings.Join(parts[:2], "::")

	return possibleServiceType
}

func associateChildren(template *TemplateStruct, resources map[string]*types.Resource) {

	for logicalId, v := range template.Resources {
		for _, child := range v.Children {
			_, ok := resources[child]
			if !ok {
				log.Infof("%s does not have parent resource", child)
				continue
			}
			log.Infof("Add child(%s) on %s", child, logicalId)

			resources[logicalId].AddChild(resources[child])
		}
	}
}

func loadLinks(template *TemplateStruct, resources map[string]*types.Resource) {

	for _, v := range template.Links {
		_, ok := resources[v.Source]
		if !ok {
			log.Warnf("Not found Source esource %s", v.Source)
			continue
		}
		source := resources[v.Source]

		_, ok = resources[v.Target]
		if !ok {
			log.Warnf("Not found Target resource %s", v.Target)
			continue
		}
		target := resources[v.Target]

		log.Infof("Add link(%s-%s)", v.Source, v.Target)
		lineWidth := v.LineWidth
		if lineWidth == 0 {
			lineWidth = 2
		}
		sourcePosition, err := types.ConvertWindrose(v.SourcePosition)
		if err != nil {
			panic(err)
		}
		targetPosition, err := types.ConvertWindrose(v.TargetPosition)
		if err != nil {
			panic(err)
		}
		link := new(types.Link).Init(source, sourcePosition, v.SourceArrowHead, target, targetPosition, v.TargetArrowHead, lineWidth)
		link.SetType(v.Type)
		resources[v.Source].AddLink(link)
		resources[v.Target].AddLink(link)
	}

}

func IsURL(str string) bool {
	if strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") {
		return true
	}
	return false
}
