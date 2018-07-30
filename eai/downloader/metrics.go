// Copyright 2018 The go-ethereumai Authors
// This file is part of the go-ethereumai library.
//
// The go-ethereumai library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereumai library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereumai library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ethereumai/go-ethereumai/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("eai/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("eai/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("eai/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("eai/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("eai/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("eai/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("eai/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("eai/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("eai/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("eai/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("eai/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("eai/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("eai/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("eai/downloader/states/drop", nil)
)
