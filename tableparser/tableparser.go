package tableparser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
)

// Struct definition

// GroupZero is first group
type GroupZero struct {
	nbDescriptionRow int
	elem             *goquery.Selection
	uid              int
	colcaption       ColCaption
	rowcaption       RowCaption
	col              []ColGroup
	row              []Row
	groupheadercell  GroupHeaderCell
	colgrp           map[int][]int
	virtualColgroup  []Cell
	colgrouphead     []ColGroup
	theadRowStack    []Row
}

// ColCaption use to caption text in col
type ColCaption struct {
	uid   int
	elem  *goquery.Selection
	etype int
	// dataset    []interface{}
}

// RowCaption use to caption text in row
type RowCaption struct {
	uid   int
	elem  *goquery.Selection
	etype int
}

// GroupHeaderCell use to group the cell in header
type GroupHeaderCell struct {
	colcaption  ColCaption
	rowcaption  RowCaption
	elem        *goquery.Selection
	description []*goquery.Selection
	caption     *goquery.Selection
	etype       int
}

// ColGroup is struct for colgroup element
type ColGroup struct {
	elem  *goquery.Selection
	uid   int
	start int
	end   int
	etype int
	level int
	col   []ColGroup
	// groupZero   GroupZero
	groupstruct []ColGroup
	header      []Cell
	headerLevel []Cell
	cell        []Cell
	dataheader  []Cell
}

// RowGroup is struct for rowgroup element
type RowGroup struct {
	elem        *goquery.Selection
	uid         int
	start       int
	end         int
	etype       int
	level       int
	row         []RowGroup
	headerlevel []Cell
	// groupZero   GroupZero
	cell              []Cell
	lastHeadingColPos int
}

// Row is struct for row
type Row struct {
	colgroup     []ColGroup
	cell         []Cell
	elem         *goquery.Selection
	uid          int
	rowpos       int
	etype        int
	level        int
	header       []Cell
	headerset    []Cell
	idsheaderset []Cell
	datacell     []Cell
	// groupZero GroupZero
}

// Cell is struct for cell
type Cell struct {
	uid        int
	rowpos     int
	colpos     int
	width      int
	height     int
	etype      int
	level      int
	start      int
	end        int
	rowlevel   int
	spanHeight int
	scope      string
	elem       *goquery.Selection
	descCell   []Cell
	row        Row
	col        ColGroup
	colgroup   ColGroup
	summary    ColGroup
	parent     ColGroup
	// describe []Cell
	keycell       []Cell
	addrowheaders []Cell
	addcolheaders []Cell
	header        []Cell
	headers       []Cell
	child         []Cell
	childs        []Cell
}

// Obj use for store global variables
type Obj struct {
	elem *goquery.Selection
}

// Variable declaration
var uidElem int
var colgroupFrame []ColGroup
var columnFrame []ColGroup
var theadRowStack []Row
var tableCellWidth int
var currentRowPos int
var spannedRow map[int]Cell
var stackRowHeader bool
var headerRowGroupCompleted bool

// Row Group Variable
var rowgroupHeaderRowStack []RowGroup
var lstRowGroup []RowGroup
var rowgroupheadercalled bool
var hassumMode bool
var tfootOnProcess bool
var previousDataHeadingColPos int
var lastHeadingSummaryColPos int
var currentRowGroup RowGroup
var currentRowGroupElement *goquery.Selection

var groupZero = GroupZero{
	nbDescriptionRow: 0,
	uid:              uidElem,
}

var obj = Obj{}

// Init function
func Init(table *goquery.Selection) error {
	// doc *goquery.Document
	// table := doc.Find("table")

	// Variable declaration
	uidElem = 0
	colgroupFrame = []ColGroup{}
	columnFrame = []ColGroup{}
	theadRowStack = []Row{}
	tableCellWidth = 0
	currentRowPos = 0
	spannedRow = map[int]Cell{}
	stackRowHeader = false
	headerRowGroupCompleted = false

	// Row Group Variable
	rowgroupHeaderRowStack = []RowGroup{}
	lstRowGroup = []RowGroup{}
	rowgroupheadercalled = false
	hassumMode = false
	tfootOnProcess = false
	lastHeadingSummaryColPos = -1

	groupZero = GroupZero{
		nbDescriptionRow: 0,
		uid:              uidElem,
	}

	obj = Obj{
		elem: table,
	}

	// Check for hassum mode
	hassumMode = table.HasClass("hassum")

	// Set the uid for the groupZero
	uidElem = uidElem + 1

	// Group Cell Header at level 0, scope=col
	groupZero.colcaption = ColCaption{
		uid:   uidElem,
		etype: 7,
	}
	uidElem = uidElem + 1

	// Group Cell Header at level 0, scope=row
	groupZero.rowcaption = RowCaption{
		uid:   uidElem,
		etype: 7,
	}
	uidElem = uidElem + 1

	groupZero.col = []ColGroup{}

	// Main Entry for the table parsing
	if table.Has("tfoot") != nil {
		tfoot := table.Find("tfoot")
		// table = table.Find("tfoot").Remove().End().ParentsUntil("~")
		table.Find("tfoot").Remove()
		table.Find("tbody").Parent().AppendSelection(tfoot)
	}

	var err error
	table.Children().EachWithBreak(func(index int, element *goquery.Selection) bool {
		var nodeName = strings.ToLower(goquery.NodeName(element))
		if nodeName == "caption" {
			err = processCaption(element)

			if err != nil {
				return false
			}
		} else if nodeName == "colgroup" {
			err = processColgroup(element, -1)

			if err != nil {
				return false
			}
		} else if nodeName == "thead" {
			currentRowGroupElement = element

			// The table should not have any row at this point
			if len(theadRowStack) != 0 || (groupZero.row != nil && len(groupZero.row) > 0) {
				err = errors.New("warning\t26\tYou can not define any row before the thead group")
				return false
			}

			stackRowHeader = true

			// This is the rowgroup header, Colgroup type can not be defined here
			element.Children().EachWithBreak(func(idx int, elem *goquery.Selection) bool {
				if strings.ToLower(goquery.NodeName(elem)) != "tr" {
					// ERROR
					err = errors.New("warning\t27\tthead element need to only have tr element as his child")
				}
				err = processRow(elem)

				if err != nil {
					return false
				}

				return true
			})

			stackRowHeader = false

			if err != nil {
				return false
			}

			// Here it"s not possible to Diggest the thead and the colgroup because we need the first data row to be half processed before
		} else if nodeName == "tbody" || nodeName == "tfoot" {
			if nodeName == "tfoot" {
				tfootOnProcess = true
			}

			// Currently there are no specific support for tfoot element, the tfoot is understood as a normal tbody
			currentRowGroupElement = element
			err = initiateRowGroup()

			if err != nil {
				return false
			}

			/*
			*
			* First tbody = data
			* All tbody with header === data
			* Subsequent tbody without header === summary
			*
			 */

			// New row group
			element.Children().EachWithBreak(func(idx int, elem *goquery.Selection) bool {
				if strings.ToLower(goquery.NodeName(elem)) != "tr" {
					// ERROR
					err = errors.New("warning\t27\tthead element need to only have tr element as his child")
					return false
				}
				err = processRow(elem)

				if err != nil {
					return false
				}

				return true
			})

			if err != nil {
				return false
			}

			err = finalizeRowGroup()

			if err != nil {
				return false
			}

			// Check for residual rowspan, there can not have cell that overflow on two or more rowgroup
			for _, span := range spannedRow {
				if span.uid != 0 && span.spanHeight > 0 {
					// That row are spanned in 2 different row group
					err = errors.New("warning\t29\tYou cannot span cell in 2 different rowgroup")
					return false
				}
			}

			spannedRow = map[int]Cell{}           /* Cleanup of any spanned row */
			rowgroupHeaderRowStack = []RowGroup{} /* Remove any rowgroup header found. */
		} else if nodeName == "tr" {
			// This are suppose to be a simple table
			err = processRow(element)

			if err != nil {
				return false
			}
		} else {
			// There is a DOM Structure error
			err = errors.New("error\t30\tUse the appropriate table markup")
			return false
		}

		return true
	})

	groupZero.theadRowStack = theadRowStack
	groupZero.colgrp = nil

	// addHeaders(groupZero)

	return err
}

func processCaption(element *goquery.Selection) error {
	groupZero.colcaption.elem = element
	groupZero.rowcaption.elem = element
	var caption *goquery.Selection
	var captionFound = false
	var description = []*goquery.Selection{}
	var groupheadercell = GroupHeaderCell{
		colcaption: groupZero.colcaption,
		rowcaption: groupZero.rowcaption,
		elem:       element,
	}

	// Extract the caption vs the description
	// There are 2 techniques,
	//	Recommanded is encapsulate the caption with "strong"
	//	Use Details/Summary element
	//	Use a simple paragraph
	if element.Children().Length() != 0 {
		// Use the contents function to retrieve the caption
		element.Contents().Each(func(index int, elem *goquery.Selection) {
			// Text node
			if caption == nil && elem.Nodes[0].Type == html.TextNode {
				// Doesn't matter what it is, but this will be
				// considered as the caption if is not empty
				var re = regexp.MustCompile(`^\s+|\s+$`)
				var captionText = re.ReplaceAllString(elem.Text(), "")
				if len(captionText) != 0 {
					caption = elem
					captionFound = true
					return
				}
				caption = nil
			} else if caption == nil && elem.Nodes[0].Type == html.ElementNode {
				// Doesn't matter what it is, the first children
				// element will be considered as the caption
				caption = elem
				return
			}
		})

		// Use the children function to retrieve the description
		element.Children().Each(func(index int, elem *goquery.Selection) {
			if captionFound {
				description = append(description, elem)
			} else {
				captionFound = true
			}
		})
	} else {
		caption = element
	}

	// Move the description in a wrapper if there is more than one element
	if len(description) >= 1 {
		groupheadercell.description = description
	}

	if caption != nil {
		groupheadercell.caption = caption
	}

	// Missing snippet
	// groupheadercell.groupZero = groupZero;

	groupheadercell.etype = 1
	groupZero.groupheadercell = groupheadercell

	// Missing snippet
	// $( elem ).data().tblparser = groupheadercell;

	return nil
}

// Pass nbvirtualcol = -1 as nil
func processColgroup(element *goquery.Selection, nbvirtualcol int) error {
	// if elem is undefined, this mean that is an big empty colgroup
	// nbvirtualcol if defined is used to create the virtual colgroup
	var colgroupspan = 0

	var colgroup = ColGroup{
		start: 0,
		end:   0,
		elem:  element,
		col:   []ColGroup{},
	}

	var width = 0

	// Missing snippet
	// if ( elem ) {
	// 	$( elem ).data().tblparser = colgroup;
	// }

	colgroup.uid = uidElem
	uidElem = uidElem + 1
	// groupZero.allParserObj = append(groupZero.allParserObj, colgroup)

	if len(colgroupFrame) != 0 {
		colgroup.start = colgroupFrame[len(colgroupFrame)-1].end + 1
	} else {
		colgroup.start = 1
	}

	// Add any exist structural col element
	if element != nil {
		element.Find("col").Each(func(index int, elem *goquery.Selection) {
			width = 1
			var spanVal, exists = elem.Attr("span")
			var col = ColGroup{
				uid:   uidElem,
				start: 0,
				end:   0,
				// groupZero: groupZero,
			}
			uidElem = uidElem + 1
			if exists == true {
				width, _ = strconv.Atoi(spanVal)
			}
			// groupZero.allParserObj = append(groupZero.allParserObj, col)
			col.start = colgroup.start + colgroupspan

			// Minus one because the default value was already calculated
			col.end = colgroup.start + colgroupspan + width - 1
			col.elem = elem
			// col.groupZero = groupZero

			// Missing snippet
			// $this.data().tblparser = col;

			colgroup.col = append(colgroup.col, col)
			columnFrame = append(columnFrame, col)
			colgroupspan = colgroupspan + width
		})
	}

	// If no col element check for the span attribute
	if len(colgroup.col) == 0 {
		if element != nil {
			width = 1
			var spanVal, exists = element.Attr("span")

			if exists == true {
				width, _ = strconv.Atoi(spanVal)
			}
		} else if nbvirtualcol != -1 {
			width = nbvirtualcol
		} else {
			return errors.New("error\t31\tInternal Error, Number of virtual column must be set [func processColgroup()]")
		}
		colgroupspan = colgroupspan + width

		// Create virtual column
		var iLen = (colgroup.start + colgroupspan)
		for i := colgroup.start; i != iLen; i++ {
			var col = ColGroup{
				uid:   uidElem,
				start: 0,
				end:   0,
			}
			uidElem = uidElem + 1
			// groupZero.allParserObj = append(groupZero.allParserObj, col)
			col.start = i
			col.end = i
			colgroup.col = append(colgroup.col, col)
			columnFrame = append(columnFrame, col)
		}
	}
	colgroup.end = colgroup.start + colgroupspan - 1
	colgroupFrame = append(colgroupFrame, colgroup)

	return nil
}

func processRowgroupHeader(colgroupHeaderColEnd int) error {
	var cell Cell
	var theadRS Row
	var theadRSNext Row
	var theadRSNextCell Cell
	var tmpStack = []Row{}
	var tmpStackCell Cell
	var tmpStackCurr Row
	var dataColgroup ColGroup
	var dataColumns []ColGroup
	var colgroup ColGroup
	var col ColGroup
	var hcolgroup ColGroup
	var gzCol ColGroup
	var currColPos int
	var currColgroupStructure []Cell
	var bigTotalColgroupFound bool

	if len(groupZero.colgrouphead) > 0 || rowgroupheadercalled == true {
		// Prevent multiple call
		return nil
	}
	rowgroupheadercalled = true
	if colgroupHeaderColEnd > 0 {
		// The first colgroup must match the colgroupHeaderColEnd
		if len(colgroupFrame) > 0 && (colgroupFrame[0].start != 1 || (colgroupFrame[0].end != colgroupHeaderColEnd && colgroupFrame[0].end != (colgroupHeaderColEnd+1))) {
			// Destroy any existing colgroup, because they are not valid
			colgroupFrame = []ColGroup{}

			return errors.New("warning\t3\tthe first colgroup must be spanned to represent the header column group")
		}
	} else {
		// This mean that are no colgroup designated to be a colgroup header
		colgroupHeaderColEnd = 0
	}

	// Associate any descriptive cell to his top header
	var iLen = len(theadRowStack)
	for i := 0; i != iLen; i++ {
		theadRS = theadRowStack[i]
		if theadRS.etype == 0 {
			theadRS.etype = 1
		}
		var jLen = len(theadRS.cell)
		for j := 0; j < jLen; j++ {
			cell = theadRowStack[i].cell[j]
			cell.scope = "col"

			// check if we have a layout cell at the top, left
			htmlstr, _ := cell.elem.Html()
			if i == 0 && j == 0 && len(htmlstr) == 0 {
				// That is a layout cell
				cell.etype = 6

				// Missing snippet
				// if ( !groupZero.layoutCell ) {
				// 	groupZero.layoutCell = []
				// }
				// groupZero.layoutCell.push( cell )

				j = cell.width - 1
				if j >= jLen {
					break
				}
			}

			// Check the next row to see if they have a corresponding description cell
			if len(theadRowStack) > i+1 {
				theadRSNext = theadRowStack[i+1]
			}

			if theadRSNext.uid != 0 {
				theadRSNextCell = theadRSNext.cell[j]
			}

			if len(cell.descCell) > 0 &&
				strings.ToLower(goquery.NodeName(cell.elem)) == "th" &&
				cell.etype != 0 &&
				theadRSNext.uid != 0 &&
				theadRSNext.uid != cell.uid &&
				theadRSNextCell.uid != 0 &&
				theadRSNextCell.etype != 0 &&
				strings.ToLower(goquery.NodeName(theadRSNextCell.elem)) == "td" &&
				theadRSNextCell.width == cell.width &&
				theadRSNextCell.height == 1 {
				// Mark the next row as a row description
				theadRSNext.etype = 5

				// Mark the cell as a cell description
				theadRSNextCell.etype = 5
				theadRSNextCell.row = theadRS
				cell.descCell = []Cell{}
				cell.descCell = append(cell.descCell, theadRSNextCell)

				// Add the description cell to the complete listing

				// Missing snippet
				// if ( !groupZero.desccell ) {
				// 	groupZero.desccell = [];
				// }
				// groupZero.desccell.push( theadRSNextCell );

				j = cell.width - 1
				if j >= jLen {
					break
				}
			}

			if cell.etype == 0 {
				cell.etype = 1
			}
		}
	}

	// Clean the theadRowStack by removing any descriptive row
	iLen = len(theadRowStack)
	for i := 0; i != iLen; i++ {
		theadRS = theadRowStack[i]
		if theadRS.etype == 5 {
			// Check if all the cell in it are set to the type 5
			var jLen = len(theadRS.cell)
			for j := 0; j != jLen; j++ {
				cell = theadRS.cell[j]
				if cell.etype != 5 && cell.etype != 6 && cell.height != 1 {
					return errors.New("warning\t4\tYou have an invalid cell inside a row description")
				}

				// Check the row before and modify their height value
				if cell.uid == theadRowStack[i-1].cell[j].uid {
					cell.height = cell.height - 1
				}
			}
			groupZero.nbDescriptionRow++
		} else {
			tmpStack = append(tmpStack, theadRS)
		}
	}

	// Array based on level as indexes for columns and group headers
	groupZero.colgrp = map[int][]int{}

	// Parser any cell in the colgroup header
	if colgroupHeaderColEnd > 0 && (len(colgroupFrame) == 1 || len(colgroupFrame) == 0) {
		// There are no colgroup elements defined.
		// All cells will be considered to be a data cells.
		// Data Colgroup
		dataColgroup = ColGroup{}
		dataColumns = []ColGroup{}
		colgroup = ColGroup{
			uid:   uidElem,
			start: colgroupHeaderColEnd + 1,
			end:   tableCellWidth,

			// Set colgroup data type
			etype: 2,
			col:   []ColGroup{},
		}
		uidElem++
		// groupZero.allParserObj = append(groupZero.allParserObj, colgroup)

		if colgroup.start > colgroup.end {
			return errors.New("warning\t5\tYou need at least one data colgroup, review your table structure")
		}

		dataColgroup = colgroup

		// Create the column
		// Create virtual column
		for i := colgroup.start; i <= colgroup.end; i++ {
			col = ColGroup{
				start: 0,
				end:   0,
				uid:   uidElem,
			}
			uidElem++
			// groupZero.allParserObj = append(groupZero.allParserObj, col)

			if groupZero.col == nil {
				groupZero.col = []ColGroup{}
			}

			col.start = i
			col.end = i
			col.groupstruct = []ColGroup{}
			col.groupstruct = append(col.groupstruct, colgroup)

			dataColumns = append(dataColumns, col)

			colgroup.col = append(colgroup.col, col)

			// Check to remove "columFrame"
			columnFrame = append(columnFrame, col)
		}

		// Default Level => 1
		groupZero.colgrp[1] = []int{}
		groupZero.colgrp[1] = append(groupZero.colgrp[1], groupZero.colcaption.etype)

		// Header Colgroup
		if colgroupHeaderColEnd > 0 {
			hcolgroup = ColGroup{
				uid:   uidElem,
				start: 1,
				end:   colgroupHeaderColEnd,
				etype: 1,
				col:   []ColGroup{},
			}
			uidElem++

			// Move to end
			// colgroupFrame = append(colgroupFrame, hcolgroup)
			// colgroupFrame = append(colgroupFrame, dataColgroup)

			// Missing snippet
			// groupZero.colcaption.dataset = dataColgroup.col

			// Create the column
			// Create virtual column
			for i := hcolgroup.start; i <= hcolgroup.end; i++ {
				col = ColGroup{
					uid:   uidElem,
					start: 0,
					end:   0,
				}
				uidElem++

				if groupZero.col == nil {
					groupZero.col = []ColGroup{}
				}

				col.start = i
				col.end = i
				col.groupstruct = []ColGroup{}
				col.groupstruct = append(col.groupstruct, hcolgroup)

				groupZero.col = append(groupZero.col, col)

				hcolgroup.col = append(hcolgroup.col, col)
				columnFrame = append(columnFrame, col)
			}

			colgroupFrame = append(colgroupFrame, hcolgroup)
			colgroupFrame = append(colgroupFrame, dataColgroup)

			for i := 0; i != len(dataColumns); i++ {
				groupZero.col = append(groupZero.col, dataColumns[i])
			}
		}

		if len(colgroupFrame) == 0 {
			colgroupFrame = append(colgroupFrame, dataColgroup)

			// Missing snippet
			// groupZero.colcaption.dataset = dataColgroup.col
		}

		// Set the header for each column
		for i := 0; i != len(groupZero.col); i++ {
			gzCol = groupZero.col[i]
			gzCol.header = []Cell{}

			for j := 0; j != len(tmpStack); j++ {
				for m := gzCol.start; m <= gzCol.end; m++ {
					if len(tmpStack[j].cell) >= m {
						cell = tmpStack[j].cell[m-1]
						if (j == 0 || (j > 0 && cell.uid != tmpStack[j-1].cell[m-1].uid)) && cell.etype == 1 {
							gzCol.header = append(gzCol.header, cell)
						}
					}
				}
			}
		}
	} else {
		// They exist colgroup element,
		//
		// -----------------------------------------------------
		//
		// Build data column group based on the data column group and summary column group.
		//
		// Suggestion: In the future, may be allow the use of a HTML5 data or CSS Option to force a colgroup to be a data group instead of a summary group
		//
		// -----------------------------------------------------
		//
		// List of real colgroup
		currColPos = 1
		if colgroupHeaderColEnd != 0 {
			// Set the current column position
			currColPos = colgroupFrame[0].end + 1
		}

		colgroup = ColGroup{
			start: currColPos,
			etype: 2,
			col:   []ColGroup{},
		}
		currColgroupStructure = []Cell{}
		bigTotalColgroupFound = false

		for _, curColgroupFrame := range colgroupFrame {
			var groupLevel = -1
			var cgrp Cell

			if bigTotalColgroupFound == true || (len(groupZero.colgrp) > 0 && len(groupZero.colgrp[0]) > 0) {
				return errors.New("error\t6\tThe Lowest column group level have been found, You may have an error in you column structure")
			}

			for _, column := range curColgroupFrame.col {
				if groupZero.col == nil {
					groupZero.col = []ColGroup{}
				}

				column.etype = 1
				column.groupstruct = []ColGroup{}
				column.groupstruct = append(column.groupstruct, curColgroupFrame)

				groupZero.col = append(groupZero.col, column)
			}

			if curColgroupFrame.start < currColPos {
				if colgroupHeaderColEnd != curColgroupFrame.end {
					return errors.New("warning\t7\tThe initial colgroup should group all the header, there are no place for any data cell")
				}

				// Skip this colgroup, this should happened only once and should represent the header colgroup

				// Assign the headers for this group
				for i := 0; i != len(curColgroupFrame.col); i++ {
					gzCol = curColgroupFrame.col[i]
					gzCol.header = []Cell{}
					for j := 0; j != len(tmpStack); j++ {
						for m := gzCol.start; m <= gzCol.end; m++ {
							if (j == 0 || (j > 0 && tmpStack[j].cell[m-1].uid != tmpStack[j-1].cell[m-1].uid)) &&
								tmpStack[j].cell[m-1].etype == 1 {
								gzCol.header = append(gzCol.header, tmpStack[j].cell[m-1])
							}
						}
					}
				}
				return nil
			}

			// get the colgroup level
			for i := 0; i != len(tmpStack); i++ {
				if len(tmpStack[i].cell) >= curColgroupFrame.end {
					tmpStackCell = tmpStack[i].cell[curColgroupFrame.end-1]
					if tmpStackCell.uid == 0 && curColgroupFrame.end > len(tmpStack[i].cell) {
						// Number of column are not corresponding to the table width
						return errors.New("warning\t3\tThe first colgroup must be spanned to represent the header column group")
					}
					if (tmpStackCell.colpos+tmpStackCell.width-1) == curColgroupFrame.end &&
						tmpStackCell.colpos >= curColgroupFrame.start {
						if groupLevel == -1 || groupLevel > (i+1) {
							// would equal at the current data cell level.
							// The lowest row level wins.
							groupLevel = i + 1
						}
					}
				} else {
					// Number of column are not corresponding to the table width
					return errors.New("warning\t3\tThe first colgroup must be spanned to represent the header column group")
				}
			}

			if groupLevel == -1 {
				// Default colgroup data Level, this happen when there
				// is no column header (same as no thead).
				groupLevel = 1
			}

			// All the cells at higher level (below the group level found)
			// of which one found, need to be inside the colgroup
			for i := groupLevel - 1; i != len(tmpStack); i++ {
				tmpStackCurr = tmpStack[i]

				// Test each cell in that group
				for j := curColgroupFrame.start - 1; j != curColgroupFrame.end; j++ {
					tmpStackCell = tmpStackCurr.cell[j]
					if tmpStackCell.colpos < curColgroupFrame.start ||
						(tmpStackCell.colpos+tmpStackCell.width-1) > curColgroupFrame.end {
						return errors.New("error\t9\tError in you header row group, there are cell that are crossing more than one colgroup")
					}
				}
			}

			// Add virtual colgroup Based on the top header
			for i := len(currColgroupStructure); i != groupLevel-1; i++ {
				tmpStackCell = tmpStack[i].cell[curColgroupFrame.start-1]

				// Use the top cell at level minus 1, that cell must be larger
				if tmpStackCell.uid != tmpStack[i].cell[curColgroupFrame.end-1].uid ||
					tmpStackCell.colpos > curColgroupFrame.start ||
					tmpStackCell.colpos+tmpStackCell.width-1 < curColgroupFrame.end {
					return errors.New("error\t10\tThe header group cell used to represent the data at level must encapsulate his group")
				}

				// Convert the header in a group header cell
				cgrp = tmpStackCell
				cgrp.level = i + 1

				cgrp.start = cgrp.colpos
				cgrp.end = cgrp.colpos + cgrp.width - 1

				// Group header cell
				cgrp.etype = 7

				currColgroupStructure = append(currColgroupStructure, cgrp)

				if groupZero.virtualColgroup == nil {
					groupZero.virtualColgroup = []Cell{}
				}
				groupZero.virtualColgroup = append(groupZero.virtualColgroup, cgrp)

				// Add the group into the level colgroup perspective
				if len(groupZero.colgrp[i+1]) == 0 {
					groupZero.colgrp[i+1] = []int{}
				}
				groupZero.colgrp[i+1] = append(groupZero.colgrp[i+1], cgrp.etype)
			}

			// Set the header list for the current group
			curColgroupFrame.header = []Cell{}
			var minusVal = 1
			if groupLevel >= 2 {
				minusVal = 2
			}
			for i := groupLevel - minusVal; i != len(tmpStack); i++ {
				for j := curColgroupFrame.start; j <= curColgroupFrame.end; j++ {
					if len(tmpStack[i].cell) >= j {
						if tmpStack[i].cell[j-1].rowpos == i+1 {
							// Attach the current colgroup to this header
							tmpStack[i].cell[j-1].colgroup = curColgroupFrame

							curColgroupFrame.header = append(curColgroupFrame.header, tmpStack[i].cell[j-1])
						}
						j = j + tmpStack[i].cell[j-1].width - 1
					}
				}
			}

			// Assign the parent header to the current header

			// Missing snippet
			// parentHeader = [];
			// for ( i = 0; i < currColgroupStructure.length - 1; i += 1 ) {
			// 	parentHeader.push( currColgroupStructure[ i ] );
			// }
			// curColgroupFrame.parentHeader = parentHeader;

			// Check to set if this group are a data group
			if len(currColgroupStructure) < groupLevel {
				// This colgroup are a data colgroup
				// The current colgroup are a data colgroup
				if curColgroupFrame.etype == 0 {
					curColgroupFrame.etype = 2

					// Set Data group type
					curColgroupFrame.level = groupLevel
				}

				// Convert ColGroup to Cell
				var curCell = Cell{
					start: curColgroupFrame.start,
					end:   curColgroupFrame.end,
					level: curColgroupFrame.level,
				}
				currColgroupStructure = append(currColgroupStructure, curCell)

				// Add the group into the level colgroup perspective
				if len(groupZero.colgrp[groupLevel]) == 0 {
					groupZero.colgrp[groupLevel] = []int{}
				}
				groupZero.colgrp[groupLevel] = append(groupZero.colgrp[groupLevel], curColgroupFrame.etype)
			}

			// Preparing the current stack for the next colgroup and set if the current are a summary group

			// Check if we need to pop out the current header colgroup
			var summaryAttached = false
			for i := len(currColgroupStructure) - 1; i != -1; i-- {
				if currColgroupStructure[i].end <= curColgroupFrame.end {
					if currColgroupStructure[i].level < groupLevel && len(theadRowStack) > 0 {
						curColgroupFrame.etype = 3
					}

					// Attach the Summary group to the colgroup
					// popped if current colgroup are type 3
					if curColgroupFrame.etype == 3 && summaryAttached == false {
						currColgroupStructure[len(currColgroupStructure)-1].summary = curColgroupFrame

						// This are used to do not attach a summary of level 4
						// to an inappropriate level 1 for example
						summaryAttached = true
					}

					var lenVal = len(currColgroupStructure)
					if lenVal > 0 {
						currColgroupStructure = currColgroupStructure[:lenVal-1]
					}
				}
			}

			if hassumMode == false {
				curColgroupFrame.etype = 2
			}

			// Catch the second and the third possible grouping at level 1
			if groupLevel == 1 && groupZero.colgrp[1] != nil && len(groupZero.colgrp[1]) > 1 && len(theadRowStack) > 0 {
				// Check if in the group at level 1 if
				// we don't already have a summary colgroup
				for i := 0; i < len(groupZero.colgrp[1]); i++ {
					if groupZero.colgrp[1][i] == 3 {
						// Congrats, we found the last possible colgroup
						curColgroupFrame.level = 0
						if len(groupZero.colgrp) > 0 && len(groupZero.colgrp[0]) == 0 {
							groupZero.colgrp[0] = []int{}
						}
						groupZero.colgrp[0] = append(groupZero.colgrp[0], curColgroupFrame.etype)

						var lenVal = len(groupZero.colgrp[1])
						if lenVal > 0 {
							groupZero.colgrp[1] = groupZero.colgrp[1][:lenVal-1]
						}

						bigTotalColgroupFound = true
						break
					}
				}

				if hassumMode == true {
					curColgroupFrame.etype = 3
				}
			}

			// Set the representative header "caption" element for a group at level 0
			// Missing snippet

			// if curColgroupFrame.level == 1 && curColgroupFrame.etype == 2 {
			// 	curColgroupFrame.repheader = "caption";
			// }

			if groupZero.col == nil {
				groupZero.col = []ColGroup{}
			}

			for _, column := range curColgroupFrame.col {
				var colpos int
				var cellWidth int
				var colHeaderLen int

				column.etype = curColgroupFrame.etype
				column.level = curColgroupFrame.level
				column.groupstruct = []ColGroup{}
				column.groupstruct = append(column.groupstruct, curColgroupFrame)
				column.header = []Cell{}

				// Find the lowest header that would represent this column
				for j := groupLevel - 1; j < len(tmpStack); j++ {
					for i := curColgroupFrame.start - 1; i < curColgroupFrame.end; i++ {
						if len(tmpStack[j].cell) > i {
							cell = tmpStack[j].cell[i]
							colpos = cell.colpos
							cellWidth = cell.width - 1
							if (colpos >= column.start && colpos <= column.end) ||
								(colpos <= column.start && colpos+cellWidth >= column.end) ||
								(colpos+cellWidth <= column.start && colpos+cellWidth >= column.end) {
								colHeaderLen = len(column.header)
								if colHeaderLen == 0 || (colHeaderLen > 0 && column.header[colHeaderLen-1].uid != cell.uid) {
									// This are the header that would represent this column
									column.header = append(column.header, cell)
									tmpStack[j].cell[i].level = curColgroupFrame.level
								}
							}
						} else {
							break
						}
					}
				}
			}
		}

		if groupZero.virtualColgroup == nil {
			groupZero.virtualColgroup = []Cell{}
		}

		// Set the Virtual Group Header Cell, if any
		for _, vGroupHeaderCell := range groupZero.virtualColgroup {
			// Set the headerLevel at the appropriate column
			for i := vGroupHeaderCell.start - 1; i < vGroupHeaderCell.end; i++ {
				if groupZero.col[i].headerLevel == nil {
					groupZero.col[i].headerLevel = []Cell{}
				}
				groupZero.col[i].headerLevel = append(groupZero.col[i].headerLevel, vGroupHeaderCell)
			}
		}
	}

	// Associate the colgroup Header in the group Zero
	if len(colgroupFrame) > 0 && colgroupHeaderColEnd > 0 {
		groupZero.colgrouphead = []ColGroup{}
		groupZero.colgrouphead = append(groupZero.colgrouphead, colgroupFrame[0])

		// Set the first colgroup type :-)
		groupZero.colgrouphead[0].etype = 1
	}

	return nil
}

func finalizeRowGroup() error {
	var err error
	// Check if the current rowgroup has been go in the rowgroup setup, if not we do
	if currentRowGroup.etype == 0 || currentRowGroup.level == 0 {
		// Colgroup Setup
		err = rowgroupSetup(false)
	}

	// If the current row group are a data group, check each row if we can found a pattern about to increment the data level for this row group
	// Update, if needed, each row and cell to take in consideration the new row group level
	// Add the row group in the groupZero Collection
	lstRowGroup = append(lstRowGroup, currentRowGroup)
	currentRowGroup = RowGroup{}

	return err
}

func initiateRowGroup() error {
	var err error
	// Finalisation of any existing row group
	if currentRowGroup.uid != 0 && currentRowGroup.etype != 0 {
		err = finalizeRowGroup()
	}

	// Initialisation of the a new row group
	currentRowGroup = RowGroup{
		elem:        currentRowGroupElement,
		row:         []RowGroup{},
		headerlevel: []Cell{},
		uid:         uidElem,
	}
	uidElem++

	return err
}

func rowgroupSetup(forceDataGroup bool) error {
	var previousRowGroup RowGroup
	var tmpHeaderLevel []Cell
	var err error

	if tfootOnProcess == true {
		currentRowGroup.etype = 3
		currentRowGroup.level = 0
		rowgroupHeaderRowStack = []RowGroup{}

		return nil
	}

	// Check if the current row group, already have some row,
	// if yes this is a new row group
	if len(rowgroupHeaderRowStack) != 0 {
		// if more than 0 cell in the stack, mark this row group as a data
		// row group and create the new row group (can be only virtual)
		if currentRowGroup.uid != 0 && currentRowGroup.etype != 0 && len(currentRowGroup.row) > 0 {
			currentRowGroupElement = nil
			err = initiateRowGroup()
			if err != nil {
				return err
			}
		}

		// We have a data row group
		currentRowGroup.etype = 2

		// Set the group header cell
		currentRowGroup.row = rowgroupHeaderRowStack
		for i := 0; i != len(rowgroupHeaderRowStack); i++ {
			rowgroupHeaderRowStack[i].cell[0].etype = 7
			rowgroupHeaderRowStack[i].cell[0].scope = "row"
			var row = Row{
				cell:  rowgroupHeaderRowStack[i].cell,
				elem:  rowgroupHeaderRowStack[i].elem,
				uid:   rowgroupHeaderRowStack[i].uid,
				etype: rowgroupHeaderRowStack[i].etype,
				level: rowgroupHeaderRowStack[i].level,
			}
			rowgroupHeaderRowStack[i].cell[0].row = row
			currentRowGroup.headerlevel = append(currentRowGroup.headerlevel, rowgroupHeaderRowStack[i].cell[0])
		}
	}

	// if no cell in the stack but first row group, mark this row group as a data row group
	if len(rowgroupHeaderRowStack) == 0 && len(lstRowGroup) == 0 {
		if currentRowGroup.etype == 1 {
			currentRowGroupElement = nil
			err = initiateRowGroup()
			if err != nil {
				return err
			}
		}

		// This is the first data row group at level 1
		currentRowGroup.etype = 2

		// Default row group level
		currentRowGroup.level = 1
	}

	// if no cell in the stack and not the first row group, this are a summary group
	// This is only valid if the first colgroup is a header colgroup.
	if len(rowgroupHeaderRowStack) == 0 && len(lstRowGroup) > 0 &&
		currentRowGroup.etype == 0 && len(colgroupFrame) > 0 && colgroupFrame[0].uid != 0 &&
		(colgroupFrame[0].etype == 1 || (colgroupFrame[0].etype == 0 && len(colgroupFrame) > 0)) &&
		forceDataGroup == false {
		currentRowGroup.etype = 3
	} else {
		currentRowGroup.etype = 2
	}

	if currentRowGroup.etype == 3 && hassumMode == false {
		currentRowGroup.etype = 2
		currentRowGroup.level = lstRowGroup[len(lstRowGroup)-1].level
	}

	// Set the Data Level for this row group
	// Calculate the appropriate row group level based on the previous rowgroup
	//	* a Summary Group decrease the row group level
	//	* a Data Group increase the row group level based of his number of row group header and the previous row group level
	//	* Dont forget to set the appropriate level to each group header cell inside this row group.
	if currentRowGroup.level == 0 {
		// Get the level of the previous group
		if len(lstRowGroup) > 0 {
			previousRowGroup = lstRowGroup[len(lstRowGroup)-1]
			if currentRowGroup.etype == 2 {
				// Data Group
				if len(currentRowGroup.headerlevel) == len(previousRowGroup.headerlevel) {
					// Same Level as the previous one
					currentRowGroup.level = previousRowGroup.level
				} else if len(currentRowGroup.headerlevel) < len(previousRowGroup.headerlevel) {
					// add the missing group heading cell
					tmpHeaderLevel = currentRowGroup.headerlevel
					currentRowGroup.headerlevel = []Cell{}

					for i := 0; i < len(previousRowGroup.headerlevel)-len(currentRowGroup.headerlevel); i++ {
						currentRowGroup.headerlevel = append(currentRowGroup.headerlevel, tmpHeaderLevel[i])
					}
					for i := 0; i < len(tmpHeaderLevel); i++ {
						currentRowGroup.headerlevel = append(currentRowGroup.headerlevel, tmpHeaderLevel[i])
					}
					currentRowGroup.level = previousRowGroup.level
				} else if len(currentRowGroup.headerlevel) > len(previousRowGroup.headerlevel) {
					// This are a new set of heading, the level equal the number of group header cell found
					currentRowGroup.level = len(currentRowGroup.headerlevel) + 1
				}
			} else if currentRowGroup.etype == 3 {
				// Summary Group
				if previousRowGroup.etype == 3 {
					currentRowGroup.level = previousRowGroup.level - 1
				} else {
					currentRowGroup.level = previousRowGroup.level
				}

				if currentRowGroup.level < 0 {
					return errors.New("warning\t12\tLast summary row group already found")
				}

				// Set the header level with the previous row group
				for i := 0; i < len(previousRowGroup.headerlevel); i++ {
					if previousRowGroup.headerlevel[i].level < currentRowGroup.level {
						currentRowGroup.headerlevel = append(currentRowGroup.headerlevel, previousRowGroup.headerlevel[i])
					}
				}
			} else {
				// Error
				// currentRowGroup.level = "Error, not calculated"
				currentRowGroup.level = -1
				return errors.New("warning\t13\tError, Row group not calculated")
			}
		} else {
			currentRowGroup.level = len(rowgroupHeaderRowStack) + 1
		}
	}

	// Ensure that each row group cell heading have their level set
	for i := 0; i < len(currentRowGroup.headerlevel); i++ {
		currentRowGroup.headerlevel[i].level = i + 1
		currentRowGroup.headerlevel[i].rowlevel = currentRowGroup.headerlevel[i].level
	}

	// reset the row header stack
	rowgroupHeaderRowStack = []RowGroup{}

	if currentRowGroup.level < 0 {
		err = errors.New("warning\t14\ttr element need to only have th or td element as his child")
	}

	return err
}

func processRow(element *goquery.Selection) error {
	// In this function there are a possible confusion about the colgroup variable name used here vs the real colgroup table,
	// In this function the colgroup is used when there are no header cell.
	currentRowPos = currentRowPos + 1
	var columnPost = 1
	var lastCellType = ""
	var lastHeadingColPos = 0
	var headingRowCell = []Cell{}
	var colKeyCell = []Cell{}
	var rowheader = Cell{}
	var isDataColgroupType = false
	var row = Row{
		colgroup: []ColGroup{}, /* === Build from colgroup object == */
		cell:     []Cell{},     /* === Build from Cell Object == */
		elem:     element,      /* Row Structure jQuery element */
		rowpos:   currentRowPos,
		uid:      uidElem,
	}
	uidElem++

	var colgroup = ColGroup{
		uid:   uidElem,
		cell:  []Cell{},
		etype: 0, /* 1 === header, 2 === data, 3 === summary, 4 === key, 5 === description, 6 === layout, 7 === group header */
	}
	uidElem++

	// Missing snippet
	// groupZero.allParserObj.push( row );
	// groupZero.allParserObj.push( colgroup );

	var err error
	// Read the row
	element.Children().Each(func(index int, elem *goquery.Selection) {
		var width = 1
		var height = 1
		var headerCell Cell
		var dataCell Cell

		var spanVal, exists = elem.Attr("colspan")
		if exists == true {
			width, _ = strconv.Atoi(spanVal)
		}

		spanVal, exists = elem.Attr("rowspan")
		if exists == true {
			height, _ = strconv.Atoi(spanVal)
		}

		switch strings.ToLower(goquery.NodeName(elem)) {
		// cell header
		case "th":
			// Check for spanned cell between cells
			fnParseSpannedRowCell(&columnPost, &lastCellType, &row, &colgroup, &lastHeadingColPos)

			headerCell = Cell{
				uid:     uidElem,
				rowpos:  currentRowPos,
				colpos:  columnPost,
				width:   width,
				height:  height,
				summary: ColGroup{},
				elem:    elem,
			}
			uidElem++

			fnPreProcessGroupHeaderCell(&colgroup, &row, &lastHeadingColPos, headerCell)

			headerCell.parent = colgroup
			headerCell.spanHeight = height - 1

			for i := 0; i < width; i++ {
				row.cell = append(row.cell, headerCell)
				spannedRow[columnPost+i] = headerCell
			}

			// Increment the column position
			columnPost = columnPost + headerCell.width
			break

		// data cell
		case "td":
			// Check for spanned cell between cells
			fnParseSpannedRowCell(&columnPost, &lastCellType, &row, &colgroup, &lastHeadingColPos)

			dataCell = Cell{
				uid:    uidElem,
				rowpos: currentRowPos,
				colpos: columnPost,
				width:  width,
				height: height,
				elem:   elem,
			}
			uidElem++

			fnPreProcessGroupDataCell(&colgroup, &row, dataCell)

			dataCell.parent = colgroup
			dataCell.spanHeight = height - 1

			for i := 0; i < width; i++ {
				row.cell = append(row.cell, dataCell)
				spannedRow[columnPost+i] = dataCell
			}

			// Increment the column position
			columnPost = columnPost + dataCell.width
			break
		default:
			err = errors.New("warning\t15\ttr element need to only have th or td element as his child")
			break
		}

		lastCellType = strings.ToLower(goquery.NodeName(elem))
	})

	if err != nil {
		return err
	}

	// Check for any spanned cell
	fnParseSpannedRowCell(&columnPost, &lastCellType, &row, &colgroup, &lastHeadingColPos)

	// Check if this the number of column for this row are equal to the other
	if tableCellWidth == 0 {
		// If not already set, we use the first row as a guideline
		tableCellWidth = len(row.cell)
	}

	if tableCellWidth != len(row.cell) {
		return errors.New("warning\t16\tThe row do not have a good width")
	}

	// Check if we are into a thead rowgroup, if yes we stop here.
	if stackRowHeader == true {
		theadRowStack = append(theadRowStack, row)
		return nil
	}

	// Add the last colgroup
	row.colgroup = append(row.colgroup, colgroup)

	//
	// Diggest the row
	//
	if lastCellType == "th" {
		// Digest the row header
		row.etype = 1

		// Check the validity of this header row
		if len(row.colgroup) == 2 && currentRowPos == 1 {
			// Check if the first is a data colgroup with only one cell
			if row.colgroup[0].etype == 2 && len(row.colgroup[0].cell) == 1 {
				// Valid row header for the row group header
				// REQUIRED: That cell need to be empty
				var htmlVal, _ = row.colgroup[0].cell[0].elem.Html()
				if len(htmlVal) == 0 {
					// We stack the row
					theadRowStack = append(theadRowStack, row)
					// We do not go further
					return nil
				}

				return errors.New("warning\t17\tThe layout cell is not empty")
			}

			// Invalid row header
			return errors.New("warning\t18\tRow group header not well structured")
		}

		if len(row.colgroup) == 1 {
			if len(row.colgroup[0].cell) > 1 {
				// this is a row associated to a header row group
				if headerRowGroupCompleted == false {
					// Good row, stack the row
					theadRowStack = append(theadRowStack, row)

					// We do not go further
					return nil
				}

				// Bad row, remove the row or split the table
				return errors.New("warning\t18\tRow group header not well structured")
			}

			if currentRowPos != 1 || row.cell[0].uid == row.cell[len(row.cell)-1].uid {
				// Stack the row found for the rowgroup header
				var rowgroup = RowGroup{
					elem:  row.elem,
					uid:   row.uid,
					etype: row.etype,
					cell:  row.cell,
				}
				rowgroupHeaderRowStack = append(rowgroupHeaderRowStack, rowgroup)

				// This will be processed on the first data row
				// End of any header row group (thead)
				headerRowGroupCompleted = true

				return nil
			}

			return errors.New("warning\t18\tRow group header not well structured")
		}

		if len(row.colgroup) > 1 && currentRowPos != 1 {
			return errors.New("warning\t21\tMove the row used as the column cell heading in the thead row group")
		}
		//
		// If Valid, process the row
		//
	} else {
		// Digest the data row or summary row
		row.etype = 2

		// This mark the end of any row group header (thead)
		headerRowGroupCompleted = true

		// Check if this row is considerated as a description row for a header
		if len(rowgroupHeaderRowStack) > 0 && row.cell[0].uid == row.cell[len(row.cell)-1].uid {
			// Horay this row are a description cell for the preceding heading
			row.etype = 5
			row.cell[0].etype = 5
			row.cell[0].row = row

			// Missing snippet
			// if ( !row.cell[ 0 ].describe ) {
			// 	row.cell[ 0 ].describe = [];
			// }

			rowgroupHeaderRowStack[len(rowgroupHeaderRowStack)-1].cell[0].descCell = []Cell{}
			rowgroupHeaderRowStack[len(rowgroupHeaderRowStack)-1].cell[0].descCell = append(rowgroupHeaderRowStack[len(rowgroupHeaderRowStack)-1].cell[0].descCell, row.cell[0])
			// row.cell[0].describe = append(row.cell[0].describe, rowgroupHeaderRowStack[len(rowgroupHeaderRowStack) - 1].cell[0])

			// Missing snippet
			// if ( !groupZero.desccell ) {
			// 	groupZero.desccell = [];
			// }
			// groupZero.desccell.push( row.cell[ 0 ] );

			// FYI - We do not push this row in any stack because this row is a description row
			// Stop the processing for this row
			return nil
		}

		//
		// Process any row used to defined the rowgroup label
		//
		if len(rowgroupHeaderRowStack) > 0 || currentRowGroup.etype == 0 {
			err = rowgroupSetup(false)
			if err != nil {
				return err
			}
		}

		row.etype = currentRowGroup.etype
		row.level = currentRowGroup.level

		if len(colgroupFrame) > 0 && colgroupFrame[0].uid > 0 && lastHeadingColPos > 0 && colgroupFrame[0].end != lastHeadingColPos && colgroupFrame[0].end == (lastHeadingColPos+1) {
			// Adjust if required, the lastHeadingColPos if colgroup are present, that would be the first colgroup
			lastHeadingColPos = lastHeadingColPos + 1
		}

		// Missing snippet
		// row.lastHeadingColPos = lastHeadingColPos

		if currentRowGroup.lastHeadingColPos == 0 {
			currentRowGroup.lastHeadingColPos = lastHeadingColPos
		}

		if previousDataHeadingColPos == 0 {
			previousDataHeadingColPos = lastHeadingColPos
		}

		// Missing snippet
		// row.rowgroup = currentRowGroup

		if currentRowGroup.lastHeadingColPos != lastHeadingColPos {
			if (lastHeadingSummaryColPos <= 0 && currentRowGroup.lastHeadingColPos < lastHeadingColPos) ||
				(lastHeadingSummaryColPos > 0 && lastHeadingSummaryColPos == lastHeadingColPos) {
				// This is a virtual summary row group
				// Check for residual rowspan, there can not have cell that overflow on two or more rowgroup
				for _, cell := range spannedRow {
					if cell.spanHeight > 0 {
						// That row are spanned in 2 different row group
						return errors.New("warning\t29\tYou cannot span cell in 2 different rowgroup")
					}
				}

				// Cleanup of any spanned row
				spannedRow = map[int]Cell{}

				// Remove any rowgroup header found.
				rowgroupHeaderRowStack = []RowGroup{}

				err = finalizeRowGroup()
				if err != nil {
					return err
				}

				currentRowGroupElement = nil

				err = initiateRowGroup()
				if err != nil {
					return err
				}

				err = rowgroupSetup(false)
				if err != nil {
					return err
				}

				// Reset the current row type
				row.etype = currentRowGroup.etype
			} else if lastHeadingSummaryColPos > 0 && previousDataHeadingColPos == lastHeadingColPos {
				// This is a virtual data row group
				// Check for residual rowspan, there can not have cell that overflow on two or more rowgroup
				for _, cell := range spannedRow {
					if cell.spanHeight > 0 {
						return errors.New("warning\t29\tYou cannot span cell in 2 different rowgroup")
					}
				}

				// Cleanup of any spanned row
				spannedRow = map[int]Cell{}

				// Remove any rowgroup header found.
				rowgroupHeaderRowStack = []RowGroup{}

				err = finalizeRowGroup()
				if err != nil {
					return err
				}

				currentRowGroupElement = nil
				err = initiateRowGroup()
				if err != nil {
					return err
				}

				err = rowgroupSetup(true)
				if err != nil {
					return err
				}

				// Reset the current row type
				row.etype = currentRowGroup.etype

				return errors.New("warning\t34\tMark properly your data row group")
			} else {
				return errors.New("warning\t32\tCheck your row cell headers structure")
			}
		}

		if currentRowGroup.lastHeadingColPos == 0 {
			currentRowGroup.lastHeadingColPos = lastHeadingColPos
		}

		if currentRowGroup.etype == 3 && lastHeadingSummaryColPos <= 0 {
			lastHeadingSummaryColPos = lastHeadingColPos
		}

		// Build the initial colgroup structure
		// If an cell header exist in that row....
		if lastHeadingColPos > 0 {
			// Process the heading colgroup associated to this row.
			headingRowCell = []Cell{}

			rowheader = Cell{}
			colKeyCell = []Cell{}

			for i := 0; i < lastHeadingColPos; i++ {
				// Check for description cell or key cell
				if strings.ToLower(goquery.NodeName(row.cell[i].elem)) == "td" {
					if i > 0 && row.cell[i].etype == 0 && row.cell[i-1].uid != 0 && len(row.cell[i-1].descCell) == 0 &&
						row.cell[i-1].etype == 1 && row.cell[i-1].height == row.cell[i].height {
						row.cell[i].etype = 5
						row.cell[i-1].descCell = []Cell{}
						row.cell[i-1].descCell = append(row.cell[i-1].descCell, row.cell[i])

						// Missing snippet

						// if len(row.cell[i].describe) == 0 {
						// 	row.cell[i].describe = []Cell{}
						// }
						// row.cell[i].describe = append(row.cell[i].describe, row.cell[i - 1])

						// if len(row.desccell) == 0 {
						// 	row.desccell = []Cell{}
						// }
						// row.desccell = append(row.desccell, row.cell[i])

						// if ( !groupZero.desccell ) {
						// 	groupZero.desccell = [];
						// }
						// groupZero.desccell.push( row.cell[ i ] );

						// Specify the scope of this description cell
						row.cell[i].scope = "row"
					}

					// Check if this cell can be an key cell associated to an cell heading
					if row.cell[i].etype == 0 {
						colKeyCell = append(colKeyCell, row.cell[i])
					}
				}

				// Set for the most appropriate header that can represent this row
				if strings.ToLower(goquery.NodeName(row.cell[i].elem)) == "th" {
					// Mark the cell to be an header cell
					row.cell[i].etype = 1
					row.cell[i].scope = "row"

					if rowheader.uid > 0 && rowheader.uid != row.cell[i].uid {
						if rowheader.height >= row.cell[i].height {
							if rowheader.height == row.cell[i].height {
								return errors.New("warning\t23\tAvoid the use of have paralel row headers, it's recommended do a cell merge to fix it")
							}

							// The current cell are a child of the previous rowheader

							// Missing snippet
							// if ( !rowheader.subheader ) {
							// 	rowheader.subheader = [];
							// 	rowheader.isgroup = true;
							// }
							// rowheader.subheader.push( row.cell[ i ] );

							// Change the current row header
							rowheader = row.cell[i]
							headingRowCell = append(headingRowCell, row.cell[i])
						} else {
							// This case are either paralel heading of growing header, this are an error.
							return errors.New("warning\t24\tFor a data row, the heading hiearchy need to be the Generic to the specific")
						}
					}

					if rowheader.uid == 0 {
						rowheader = row.cell[i]
						headingRowCell = append(headingRowCell, row.cell[i])
					}

					for j := 0; j < len(colKeyCell); j++ {
						if colKeyCell[j].etype == 0 && len(row.cell[i].keycell) == 0 && colKeyCell[j].height == row.cell[i].height {
							colKeyCell[j].etype = 4
							row.cell[i].keycell = []Cell{}
							row.cell[i].keycell = append(row.cell[i].keycell, colKeyCell[j])
						}
					}
				}
			}

			// All the cell that have no "type" in the colKeyCell collection are problematic cells
			for _, cell := range colKeyCell {
				if cell.etype == 0 {
					return errors.New("warning\t25\tYou have a problematic key cell")
				}
			}

			row.header = headingRowCell
		} else {
			// There are only at least one colgroup,
			// Any colgroup tag defined but be equal or greater than 0.
			// if colgroup tag defined, they are all data colgroup.
			lastHeadingColPos = 0

			if len(colgroupFrame) == 0 {
				err = processColgroup(nil, tableCellWidth)
				if err != nil {
					return err
				}
			}
		}

		//
		// Process the table row heading and colgroup if required
		//
		err = processRowgroupHeader(lastHeadingColPos)

		if err != nil {
			return err
		}

		if currentRowGroup.headerlevel == nil {
			row.headerset = []Cell{}
		} else {
			row.headerset = currentRowGroup.headerlevel
		}

		if lastHeadingColPos != 0 {
			lastHeadingColPos = colgroupFrame[0].end /* colgroupFrame must be defined here */
		}

		//
		// Associate the data cell type with the colgroup if any,
		// Process the data cell. There are a need to have at least one data cell per data row.
		if row.datacell == nil {
			row.datacell = []Cell{}
		}

		for i := lastHeadingColPos; i < len(row.cell); i++ {
			isDataColgroupType = true

			var tempj = 1
			if lastHeadingColPos == 0 {
				tempj = 0
			}
			for j := tempj; j < len(colgroupFrame); j++ {
				// If colgroup, the first are always header colgroup
				if colgroupFrame[j].start <= row.cell[i].colpos && row.cell[i].colpos <= colgroupFrame[j].end {
					if row.etype == 3 || colgroupFrame[j].etype == 3 {
						row.cell[i].etype = 3 /* Summary Cell */
					} else {
						row.cell[i].etype = 2
					}

					// Test if this cell is a layout cell
					if row.etype == 3 && colgroupFrame[j].etype == 3 && len(row.cell[i].elem.Text()) == 0 {
						row.cell[i].etype = 6
					}
				}
				isDataColgroupType = !isDataColgroupType
			}

			if len(colgroupFrame) == 0 {
				// There are no colgroup definition, this cell are set to be a datacell
				row.cell[i].etype = 2
			}

			// Add row header when the cell is span into more than one row
			if row.cell[i].rowpos < currentRowPos {
				if row.cell[i].addrowheaders == nil {
					// addrowheaders for additional row headers
					row.cell[i].addrowheaders = []Cell{}
				}
				if len(row.header) > 0 {
					for j := 0; j < len(row.header); j++ {
						if (row.header[j].rowpos == currentRowPos && len(row.cell[i].addrowheaders) == 0) ||
							(row.header[j].rowpos == currentRowPos && row.cell[i].addrowheaders[len(row.cell[i].addrowheaders)-1].uid != row.header[j].uid) {
							// Add the current header
							row.cell[i].addrowheaders = append(row.cell[i].addrowheaders, row.header[j])
						}
					}
				}
			}
		}

		// Add the cell in his appropriate column
		if groupZero.col == nil {
			groupZero.col = []ColGroup{}
		}

		for i := 0; i < len(groupZero.col); i++ {
			for j := groupZero.col[i].start - 1; j < groupZero.col[i].end; j++ {
				if groupZero.col[i].cell == nil {
					groupZero.col[i].cell = []Cell{}
				}

				// Be sure to do not include twice the same cell for a column spanned in 2 or more column
				if !(j > groupZero.col[i].start-1 && groupZero.col[i].cell[len(groupZero.col[i].cell)-1].uid == row.cell[j].uid) {
					if len(row.cell) > j {
						if row.cell[j].uid != 0 {
							if row.cell[j].col.uid == 0 {
								row.cell[j].col = groupZero.col[i]
							}
							groupZero.col[i].cell = append(groupZero.col[i].cell, row.cell[j])
						} else {
							return errors.New("warning\t35\tColumn, col element, are not correctly defined")
						}
					}
				}
			}
		}

		// Associate the row with the cell and Colgroup/Col association
		for i := 0; i < len(row.cell); i++ {
			if row.cell[i].row.uid == 0 {
				row.cell[i].row = row
			}
			row.cell[i].rowlevel = currentRowGroup.level

			// Missing snippet
			// row.cell[ i ].rowlevelheader = currentRowGroup.headerlevel
			// row.cell[ i ].rowgroup = currentRowGroup

			if i > 0 && row.cell[i-1].uid == row.cell[i].uid && row.cell[i].etype != 1 && row.cell[i].etype != 5 &&
				row.cell[i].rowpos == currentRowPos && row.cell[i].colpos <= i {
				if row.cell[i].addcolheaders == nil {
					// addcolheaders for additional col headers
					row.cell[i].addcolheaders = []Cell{}
				}

				// Add the column header if required
				if groupZero.col[i].uid != 0 && len(groupZero.col[i].header) > 0 {
					for j := 0; j < len(groupZero.col[i].header); j++ {
						if groupZero.col[i].header[j].colpos == i+1 {
							// Add the current header
							row.cell[i].addcolheaders = append(row.cell[i].addcolheaders, groupZero.col[i].header[j])
						}
					}
				}
			}
		}
	}

	row.colgroup = nil

	// Add the row to the groupZero
	if groupZero.row == nil {
		groupZero.row = []Row{}
	}
	groupZero.row = append(groupZero.row, row)

	var rowgroup = RowGroup{
		uid:   row.uid,
		etype: row.etype,
		level: row.level,
		elem:  row.elem,
		cell:  row.cell,
	}
	currentRowGroup.row = append(currentRowGroup.row, rowgroup)

	return nil
}

// Add headers information to the table parsed data structure
// Similar sample of code as the HTML Table validator
func addHeaders(tblparser GroupZero) {
	var headStackLength = len(tblparser.theadRowStack)
	var currRow = Row{}
	var currCell = Cell{}
	var coldataheader []Cell
	var rowheaders []Cell
	var rowheadersgroup []Cell
	var currrowheader []Cell
	var currCol = ColGroup{}
	var colheaders []Cell
	var colheadersgroup []Cell
	var ongoingRowHeader []Cell

	// Set ID and Header for the table head
	for i := 0; i < headStackLength; i++ {
		currRow = tblparser.theadRowStack[i]
		for j := 0; j < len(currRow.cell); j++ {
			currCell = currRow.cell[j]
			if (currCell.etype == 1 || currCell.etype == 7) &&
				(!(j > 0 && currCell.uid == currRow.cell[j-1].uid) &&
					!(i > 0 && currCell.uid == tblparser.theadRowStack[i-1].cell[j].uid)) {
				// Imediate header
				if currCell.header == nil {
					currCell.header = []Cell{}
				}

				// all the headers
				if currCell.headers == nil {
					currCell.headers = []Cell{}
				}

				// Imediate sub cell
				if currCell.child == nil {
					currCell.child = []Cell{}
				}

				// Imediate sub cell
				if currCell.childs == nil {
					currCell.childs = []Cell{}
				}

				// Set the header of the current cell if required
				if i > 0 {
					// All the header cells
					var cellHeaderLength = len(tblparser.theadRowStack[i-1].cell[j].header)
					for k := 0; k < cellHeaderLength; k++ {
						currCell.headers = append(currCell.headers, tblparser.theadRowStack[i-1].cell[j].header[k])
						tblparser.theadRowStack[i-1].cell[j].header[k].childs = append(tblparser.theadRowStack[i-1].cell[j].header[k].childs, currCell)
					}

					// Imediate header cell
					currCell.header = append(currCell.header, tblparser.theadRowStack[i-1].cell[j])
					currCell.headers = append(currCell.headers, tblparser.theadRowStack[i-1].cell[j])
					tblparser.theadRowStack[i-1].cell[j].child = append(tblparser.theadRowStack[i-1].cell[j].child, currCell)
				}

				// Set the header on his descriptive cell if any
				if currCell.descCell != nil && len(currCell.descCell) > 0 {
					currCell.descCell[0].header = []Cell{}
					currCell.descCell[0].headers = []Cell{}
					currCell.descCell[0].header = append(currCell.descCell[0].header, currCell)
					currCell.descCell[0].headers = append(currCell.descCell[0].headers, currCell)
				}
			}
		}
	}

	// Set Id/headers for header cell and data cell in the table.
	for i := 0; i < len(tblparser.row); i++ {
		currRow = tblparser.row[i]
		rowheaders = []Cell{}
		rowheadersgroup = []Cell{}
		currrowheader = []Cell{}
		coldataheader = []Cell{}
		ongoingRowHeader = []Cell{}

		// Get or Generate a unique ID for each header in this row
		if len(currRow.headerset) > 0 && len(currRow.idsheaderset) == 0 {
			for j := 0; j < len(currRow.headerset); j++ {
				rowheadersgroup = append(rowheadersgroup, currRow.headerset[j])
			}
			currRow.idsheaderset = rowheadersgroup
		}

		if len(currRow.header) > 0 {
			for j := 0; j < len(currRow.header); j++ {
				rowheaders = append(rowheaders, currRow.header[j])
			}
		}

		rowheaders = append(currRow.idsheaderset, rowheaders...)

		for j := 0; j < len(currRow.cell); j++ {
			if j == 0 || (j > 0 && currRow.cell[j].uid != currRow.cell[j-1].uid) {
				currCell = currRow.cell[j]
				coldataheader = []Cell{}

				// Imediate header
				if currCell.header == nil {
					currCell.header = []Cell{}
				}

				// all the headers
				if currCell.headers == nil {
					currCell.headers = []Cell{}
				}

				if currCell.col.uid != 0 && len(currCell.col.dataheader) == 0 {
					currCol = currCell.col
					colheaders = []Cell{}
					colheadersgroup = []Cell{}
					if len(currCol.headerLevel) > 0 {
						for m := 0; m < len(currCol.headerLevel); m++ {
							colheadersgroup = append(colheadersgroup, currCol.headerLevel[m])
						}
					}

					if len(currCol.header) > 0 {
						for m := 0; m < len(currCol.header); m++ {
							colheaders = append(colheaders, currCol.header[m])
						}
					}

					if currCol.dataheader == nil {
						currCol.dataheader = []Cell{}
					}

					currCol.dataheader = append(currCol.dataheader, colheadersgroup...)
					currCol.dataheader = append(currCol.dataheader, colheaders...)
				}

				if currCell.col.uid != 0 && len(currCell.col.dataheader) > 0 {
					coldataheader = currCell.col.dataheader
				}

				if currCell.etype == 1 {
					// Imediate sub cell
					if currCell.child == nil {
						currCell.child = []Cell{}
					}

					// All the sub cell
					if currCell.childs == nil {
						currCell.childs = []Cell{}
					}

					for m := 0; m < len(ongoingRowHeader); m++ {
						if currCell.colpos == ongoingRowHeader[m].colpos+ongoingRowHeader[m].width {
							var childLength = len(ongoingRowHeader[m].child)
							if childLength == 0 || (childLength > 0 && ongoingRowHeader[m].child[childLength-1].uid != currCell.uid) {
								ongoingRowHeader[m].child = append(ongoingRowHeader[m].child, currCell)
							}
						}
						ongoingRowHeader[m].childs = append(ongoingRowHeader[m].childs, currCell)
					}

					for m := 0; m < len(currRow.idsheaderset); m++ {
						// All the sub cell
						if currRow.idsheaderset[m].childs == nil {
							currRow.idsheaderset[m].childs = []Cell{}
						}
						currRow.idsheaderset[m].childs = append(currRow.idsheaderset[m].childs, currCell)
					}

					currCell.header = append(currCell.header, ongoingRowHeader...)
					currCell.headers = append(currCell.headers, coldataheader...)
					currCell.headers = append(currCell.headers, currRow.idsheaderset...)
					currCell.headers = append(currCell.headers, ongoingRowHeader...)

					ongoingRowHeader = append(ongoingRowHeader, currCell)
				}

				if currCell.etype == 2 || currCell.etype == 3 {
					// Get Current Column Headers
					currrowheader = rowheaders
					if len(currCell.addcolheaders) > 0 {
						for m := 0; m < len(currCell.addcolheaders); m++ {
							coldataheader = append(coldataheader, currCell.addcolheaders[m])
						}
					}

					if len(currCell.addrowheaders) > 0 {
						for m := 0; m < len(currCell.addrowheaders); m++ {
							currrowheader = append(currrowheader, currCell.addrowheaders[m])
						}
					}

					currCell.headers = append(currCell.headers, coldataheader...)
					currCell.headers = append(currCell.headers, currrowheader...)
					currCell.header = currCell.headers
				}
			}
		}
	}
} /* END addHeaders function*/

func fnPreProcessGroupHeaderCell(colgroup *ColGroup, row *Row, lastHeadingColPos *int, headerCell Cell) {
	if colgroup.etype == 0 {
		colgroup.etype = 1
	}
	if colgroup.etype != 1 {
		// Creation of a new colgroup
		// Add the previous colgroup
		row.colgroup = append(row.colgroup, *colgroup)

		// Create a new colgroup
		*colgroup = ColGroup{
			uid:   uidElem,
			etype: 1,
			cell:  []Cell{},
		}
		uidElem++
	}
	colgroup.cell = append(colgroup.cell, headerCell)
	*lastHeadingColPos = headerCell.colpos + headerCell.width - 1
}

func fnPreProcessGroupDataCell(colgroup *ColGroup, row *Row, dataCell Cell) {
	if colgroup.etype == 0 {
		colgroup.etype = 2
	}

	// Check if we need to create a summary colgroup (Based on the top colgroup definition)
	if colgroup.etype != 2 {
		// Creation of a new colgroup
		// Add the previous colgroup
		row.colgroup = append(row.colgroup, *colgroup)

		// Create a new colgroup
		*colgroup = ColGroup{
			uid:   uidElem,
			etype: 2,
			cell:  []Cell{},
		}
		uidElem++
	}
	colgroup.cell = append(colgroup.cell, dataCell)
}

func fnParseSpannedRowCell(columnPos *int, lastCellType *string, row *Row, colgroup *ColGroup, lastHeadingColPos *int) {
	var currCell Cell

	// Check for spanned row
	for *columnPos <= tableCellWidth {
		if spannedRow[*columnPos].uid == 0 {
			break
		}
		currCell = spannedRow[*columnPos]

		if currCell.spanHeight > 0 && currCell.colpos == *columnPos {
			if currCell.height+currCell.rowpos-currCell.spanHeight != currentRowPos {
				break
			}
			*lastCellType = strings.ToLower(goquery.NodeName(currCell.elem))

			if *lastCellType == "th" {
				fnPreProcessGroupHeaderCell(colgroup, row, lastHeadingColPos, currCell)
			} else if *lastCellType == "td" {
				fnPreProcessGroupDataCell(colgroup, row, currCell)
			}

			// Adjust the spanned value for the next check
			if currCell.spanHeight == 1 {
				currCell.spanHeight = 0
			} else {
				currCell.spanHeight = currCell.spanHeight - 1
			}

			// In javascript, change a property of an object referenced by a variable
			// does change the underlying object. So in Go we must assign it again
			spannedRow[*columnPos] = currCell

			for j := 0; j < currCell.width; j++ {
				row.cell = append(row.cell, currCell)
			}

			// Increment the column position
			*columnPos = *columnPos + currCell.width
		} else {
			break
		}
	}
}
