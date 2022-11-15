// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("g/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("g/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("g/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("g/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("g/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("g/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("g/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("g/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("g/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("g/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("g/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("g/downloader/receipts/timeout", nil)

	throttleCounter = metrics.NewRegisteredCounter("g/downloader/throttle", nil)
)
