package filter

import "strings"

const (
	anySelect = "$any"
	empty     = "$empty"
)

type Tag struct {
	// execute the scene
	SelectScene string
	//Does this field need to be ignored?
	IsOmitField bool
	// is the selected case, indicating whether the field needs to be parsed
	IsSelect bool
	//Field Name
	UseFieldName string
	//IsAnonymous identifies whether the field is an anonymous field
	IsAnonymous bool
	////Ignore if empty
	Omitempty bool
}

func newSelectTag(tag, selectScene, fieldName string) Tag {

	tagEl := Tag{
		SelectScene: selectScene,
		IsOmitField: true,
	}
	tags := strings.Split(tag, ",")
	tagEl.UseFieldName = fieldName

	if len(tags) < 2 {
		return tagEl
	} else {
		if tags[0] == "" {
			tagEl.IsAnonymous = true
		} else {
			tagEl.UseFieldName = tags[0]
		}
	}
	if tags[1] == "omitempty" {
		tagEl.Omitempty = true
	}

	for _, s := range tags {
		if strings.HasPrefix(s, "select(") {
			selectStr := s[7 : len(s)-1]
			scene := strings.Split(selectStr, "|")
			for _, v := range scene {
				if v == selectScene || v == anySelect {
					//说明选中了tag里的场景,不应该被忽略
					tagEl.IsOmitField = false
					tagEl.IsSelect = true
					return tagEl
				}
			}
		}
	}
	return tagEl
}

func newOmitTag(tag, omitScene, fieldName string) Tag {
	tagEl := Tag{
		SelectScene: omitScene,
		IsOmitField: false,
		IsSelect:    true,
	}
	tags := strings.Split(tag, ",")
	tagEl.UseFieldName = fieldName

	if len(tags) < 2 {
		if len(tags) == 1 {
			if tags[0] != "" {
				tagEl.UseFieldName = tags[0]
			}
		}
		return tagEl
	} else {
		if tags[0] == "" {
			tagEl.IsAnonymous = true
		} else {
			tagEl.UseFieldName = tags[0]
		}
	}
	if tags[1] == "omitempty" {
		tagEl.Omitempty = true
	}

	for _, s := range tags {
		if strings.HasPrefix(s, "omit(") {
			selectStr := s[5 : len(s)-1]
			scene := strings.Split(selectStr, "|")
			for _, v := range scene {
				if v == omitScene || v == anySelect {
					//说明选中了tag里的场景,应该被忽略
					tagEl.IsOmitField = true
					tagEl.IsSelect = false
					return tagEl
				}
			}
		}
	}
	return tagEl
}

func newOmitNotTag(omitScene, fieldName string) Tag {
	return Tag{
		IsSelect:     true,
		UseFieldName: fieldName,
		SelectScene:  omitScene,
	}
}
