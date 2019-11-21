package main

import (
	"github.com/jedib0t/go-pretty/table"
	"os"
	"time"
)

type GuiDescription struct {
	MenuItems []struct {
		Name           string        `json:"Name"`
		SortID         string        `json:"SortId"`
		SubMenuEntries []interface{} `json:"SubMenuEntries"`
		ParameterNode  bool          `json:"ParameterNode"`
		ImageName      string        `json:"ImageName"`
		TabViews       []struct {
			IsExpertView         bool   `json:"IsExpertView"`
			TabName              string `json:"TabName"`
			GuiID                int    `json:"GuiId"`
			BundleID             int    `json:"BundleId"`
			ParameterDescriptors []struct {
				ValueID               int64  `json:"ValueId"`
				SortID                int    `json:"SortId"`
				SubBundleID           int    `json:"SubBundleId"`
				ParameterID           int    `json:"ParameterId"`
				IsReadOnly            bool   `json:"IsReadOnly"`
				NoDataPoint           bool   `json:"NoDataPoint"`
				IsExpertProtectable   bool   `json:"IsExpertProtectable"`
				Name                  string `json:"Name"`
				Group                 string `json:"Group"`
				ProtGrp               string `json:"ProtGrp,omitempty"`
				ControlType           int    `json:"ControlType"`
				Value                 string `json:"Value"`
				ValueState            int    `json:"ValueState"`
				HasDependentParameter bool   `json:"HasDependentParameter"`
				Unit                  string `json:"Unit,omitempty"`
				Decimals              int    `json:"Decimals,omitempty"`
				ListItems             []struct {
					Value               string `json:"Value"`
					DisplayText         string `json:"DisplayText"`
					IsSelectable        bool   `json:"IsSelectable"`
					HighlightIfSelected bool   `json:"HighlightIfSelected"`
				} `json:"ListItems,omitempty"`
				MinValueCondition         string  `json:"MinValueCondition,omitempty"`
				MaxValueCondition         string  `json:"MaxValueCondition,omitempty"`
				MinValue                  float64 `json:"MinValue,omitempty"`
				MaxValue                  float64 `json:"MaxValue,omitempty"`
				StepWidth                 float64 `json:"StepWidth,omitempty"`
				ChildParameterDescriptors []struct {
					ValueID                   int    `json:"ValueId"`
					SortID                    int    `json:"SortId"`
					SubBundleID               int    `json:"SubBundleId"`
					ParameterID               int    `json:"ParameterId"`
					IsReadOnly                bool   `json:"IsReadOnly"`
					NoDataPoint               bool   `json:"NoDataPoint"`
					IsExpertProtectable       bool   `json:"IsExpertProtectable"`
					Name                      string `json:"Name"`
					ControlType               int    `json:"ControlType"`
					ValueState                int    `json:"ValueState"`
					HasDependentParameter     bool   `json:"HasDependentParameter"`
					ChildParameterDescriptors []struct {
						ValueID               int64   `json:"ValueId"`
						SortID                int     `json:"SortId"`
						SubBundleID           int     `json:"SubBundleId"`
						ParameterID           int64   `json:"ParameterId"`
						IsReadOnly            bool    `json:"IsReadOnly"`
						NoDataPoint           bool    `json:"NoDataPoint"`
						IsExpertProtectable   bool    `json:"IsExpertProtectable"`
						Name                  string  `json:"Name"`
						Group                 string  `json:"Group"`
						ControlType           int     `json:"ControlType"`
						ValueState            int     `json:"ValueState"`
						HasDependentParameter bool    `json:"HasDependentParameter"`
						Unit                  string  `json:"Unit"`
						MinValueCondition     string  `json:"MinValueCondition"`
						MaxValueCondition     string  `json:"MaxValueCondition"`
						MinValue              float64 `json:"MinValue"`
						MaxValue              float64 `json:"MaxValue"`
						StepWidth             float64 `json:"StepWidth"`
						Decimals              int     `json:"Decimals"`
					} `json:"ChildParameterDescriptors"`
				} `json:"ChildParameterDescriptors,omitempty"`
			} `json:"ParameterDescriptors"`
			ViewType           int       `json:"ViewType"`
			SvgSchemaDeviceID  int       `json:"SvgSchemaDeviceId"`
			GetValueLastAccess time.Time `json:"GetValueLastAccess"`
			TabViewGroups      []struct {
				GroupName       string `json:"GroupName"`
				IsTitleEditable bool   `json:"IsTitleEditable"`
			} `json:"TabViewGroups"`
		} `json:"TabViews"`
	} `json:"MenuItems"`
	DynFaultMessageDevices     []interface{} `json:"DynFaultMessageDevices"`
	SystemHasWRSClassicDevices bool          `json:"SystemHasWRSClassicDevices"`
}

func printGuiParameters(d GuiDescription) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ValueID","Name", "Group", "Unit", "Value","Options"})
	for _,menuItem := range d.MenuItems {
		for _,tabView := range menuItem.TabViews {
			for _,parameterDescriptor := range tabView.ParameterDescriptors {
				t.AppendRow(table.Row{
					parameterDescriptor.ValueID,
					parameterDescriptor.Name,
					parameterDescriptor.Group,
					parameterDescriptor.Unit,
					parameterDescriptor.Value,
					"",
				})
				if (parameterDescriptor.ListItems!=nil) {
					for _,listItem :=range parameterDescriptor.ListItems {
						t.AppendRow(table.Row{
							"-->",
							parameterDescriptor.Name,
							"-",
							"-",
							"-",
							listItem.Value + "-> " + listItem.DisplayText,

						})
					}
				}
 			}
		}
	}
	//t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
	t.SetStyle(table.StyleLight)
	t.Render()
}
