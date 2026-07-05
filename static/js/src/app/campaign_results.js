var map = null
var doPoll = true;

// statuses is a helper map to point result statuses to ui classes
var statuses = {
    "Email Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "Emails Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "In progress": {
        label: "label-primary"
    },
    "Queued": {
        label: "label-info"
    },
    "Completed": {
        label: "label-success"
    },
    "Email Opened": {
        color: "#f9bf3b",
        label: "label-warning",
        icon: "fa-envelope-open",
        point: "ct-point-opened"
    },
    "Clicked Link": {
        color: "#F39C12",
        label: "label-clicked",
        icon: "fa-mouse-pointer",
        point: "ct-point-clicked"
    },
    "Success": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    //not a status, but is used for the campaign timeline and user timeline
    "Email Reported": {
        color: "#45d6ef",
        label: "label-info",
        icon: "fa-bullhorn",
        point: "ct-point-reported"
    },
    "Error": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Error Sending Email": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Submitted Data": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Unknown": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-question",
        point: "ct-point-error"
    },
    "Sending": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-spinner",
        point: "ct-point-sending"
    },
    "Retrying": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-clock-o",
        point: "ct-point-error"
    },
    "Scheduled": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-clock-o",
        point: "ct-point-sending"
    },
    "Campaign Created": {
        label: "label-success",
        icon: "fa-rocket"
    }
}

var statusMapping = {
    "Email Sent": "sent",
    "Email Opened": "opened",
    "Clicked Link": "clicked",
    "Submitted Data": "submitted_data",
    "Email Reported": "reported",
}

// This is an underwhelming attempt at an enum
// until I have time to refactor this appropriately.
var progressListing = [
    "Email Sent",
    "Email Opened",
    "Clicked Link",
    "Submitted Data"
]

var campaign = {}
var bubbles = []
var scheduleTable = null
var scheduleMetadata = {
    scenarios: {},
    templates: {}
}

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#modal").modal('hide')
    $("#resultsTable").dataTable().DataTable().clear().draw()
}

// Deletes a campaign after prompting the user
function deleteCampaign() {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the campaign. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete Campaign",
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.delete(campaign.id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            }).catch(function (error) {
                Swal.showValidationMessage(
                    `Request failed: ${error}`
                );
                return false;
            })
        }
    }).then(function (result) {
        if(result.value){
            Swal.fire(
                'Campaign Deleted!',
                'This campaign has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.href = '/campaigns'
        })
    })
}

// Completes a campaign after prompting the user
function completeCampaign() {
    Swal.fire({
        title: "Are you sure?",
        text: "Gophish will stop processing events for this campaign",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Complete Campaign",
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.complete(campaign.id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value){
            Swal.fire(
                'Campaign Completed!',
                'This campaign has been completed!',
                'success'
            );
            $('#complete_button')[0].disabled = true;
            $('#complete_button').text('Completed!')
            doPoll = false;
        }
    })
}

// Exports campaign results as a CSV file
function exportAsCSV(scope) {
    exportHTML = $("#exportButton").html()
    var csvScope = null
    var filename = campaign.name + ' - ' + capitalize(scope) + '.csv'
    switch (scope) {
        case "results":
            csvScope = campaign.results
            break;
        case "events":
            csvScope = campaign.timeline
            break;
    }
    if (!csvScope) {
        return
    }
    $("#exportButton").html('<i class="fa fa-spinner fa-spin"></i>')
    var csvString = Papa.unparse(csvScope, {
        'escapeFormulae': true
    })
    var csvData = new Blob([csvString], {
        type: 'text/csv;charset=utf-8;'
    });
    if (navigator.msSaveBlob) {
        navigator.msSaveBlob(csvData, filename);
    } else {
        var csvURL = window.URL.createObjectURL(csvData);
        var dlLink = document.createElement('a');
        dlLink.href = csvURL;
        dlLink.setAttribute('download', filename)
        document.body.appendChild(dlLink)
        dlLink.click();
        document.body.removeChild(dlLink)
    }
    $("#exportButton").html(exportHTML)
}

function replay(event_idx) {
    request = campaign.timeline[event_idx]
    details = JSON.parse(request.details)
    url = null
    form = $('<form>').attr({
        method: 'POST',
        target: '_blank',
    })
    /* Create a form object and submit it */
    $.each(Object.keys(details.payload), function (i, param) {
        if (param == "rid") {
            return true;
        }
        if (param == "__original_url") {
            url = details.payload[param];
            return true;
        }
        $('<input>').attr({
            name: param,
        }).val(details.payload[param]).appendTo(form);
    })
    /* Ensure we know where to send the user */
    // Prompt for the URL
    Swal.fire({
        title: 'Where do you want the credentials submitted to?',
        input: 'text',
        showCancelButton: true,
        inputPlaceholder: "http://example.com/login",
        inputValue: url || "",
        inputValidator: function (value) {
            return new Promise(function (resolve, reject) {
                if (value) {
                    resolve();
                } else {
                    reject('Invalid URL.');
                }
            });
        }
    }).then(function (result) {
        if (result.value){
            url = result.value
            submitForm()
        }
    })
    return
    submitForm()

    function submitForm() {
        form.attr({
            action: url
        })
        form.appendTo('body').submit().remove()
    }
}

/**
 * Returns an HTML string that displays the OS and browser that clicked the link
 * or submitted credentials.
 *
 * @param {object} event_details - The "details" parameter for a campaign
 *  timeline event
 *
 */
var renderDevice = function (event_details) {
    var ua = UAParser(details.browser['user-agent'])
    var detailsString = '<div class="timeline-device-details">'

    var deviceIcon = 'laptop'
    if (ua.device.type) {
        if (ua.device.type == 'tablet' || ua.device.type == 'mobile') {
            deviceIcon = ua.device.type
        }
    }

    var deviceVendor = ''
    if (ua.device.vendor) {
        deviceVendor = ua.device.vendor.toLowerCase()
        if (deviceVendor == 'microsoft') deviceVendor = 'windows'
    }

    var deviceName = 'Unknown'
    if (ua.os.name) {
        deviceName = ua.os.name
        if (deviceName == "Mac OS") {
            deviceVendor = 'apple'
        } else if (deviceName == "Windows") {
            deviceVendor = 'windows'
        }
        if (ua.device.vendor && ua.device.model) {
            deviceName = ua.device.vendor + ' ' + ua.device.model
        }
    }

    if (ua.os.version) {
        deviceName = deviceName + ' (OS Version: ' + ua.os.version + ')'
    }

    deviceString = '<div class="timeline-device-os"><span class="fa fa-stack">' +
        '<i class="fa fa-' + escapeHtml(deviceIcon) + ' fa-stack-2x"></i>' +
        '<i class="fa fa-vendor-icon fa-' + escapeHtml(deviceVendor) + ' fa-stack-1x"></i>' +
        '</span> ' + escapeHtml(deviceName) + '</div>'

    detailsString += deviceString

    var deviceBrowser = 'Unknown'
    var browserIcon = 'info-circle'
    var browserVersion = ''

    if (ua.browser && ua.browser.name) {
        deviceBrowser = ua.browser.name
        // Handle the "mobile safari" case
        deviceBrowser = deviceBrowser.replace('Mobile ', '')
        if (deviceBrowser) {
            browserIcon = deviceBrowser.toLowerCase()
            if (browserIcon == 'ie') browserIcon = 'internet-explorer'
        }
        browserVersion = '(Version: ' + ua.browser.version + ')'
    }

    var browserString = '<div class="timeline-device-browser"><span class="fa fa-stack">' +
        '<i class="fa fa-' + escapeHtml(browserIcon) + ' fa-stack-1x"></i></span> ' +
        deviceBrowser + ' ' + browserVersion + '</div>'

    detailsString += browserString
    detailsString += '</div>'
    return detailsString
}

function renderTimeline(data) {
    record = {
        "id": data[0],
        "first_name": data[2],
        "last_name": data[3],
        "email": data[4],
        "position": data[5],
        "status": data[6],
        "reported": data[7],
        "send_date": data[8]
    }
    results = '<div class="timeline col-sm-12 well well-lg">' +
        '<h6>Timeline for ' + escapeHtml(record.first_name) + ' ' + escapeHtml(record.last_name) +
        '</h6><span class="subtitle">Email: ' + escapeHtml(record.email) +
        '<br>Result ID: ' + escapeHtml(record.id) + '</span>' +
        '<div class="timeline-graph col-sm-6">'
    $.each(campaign.timeline, function (i, event) {
        if (!event.email || event.email == record.email) {
            // Add the event
            results += '<div class="timeline-entry">' +
                '    <div class="timeline-bar"></div>'
            results +=
                '    <div class="timeline-icon ' + statuses[event.message].label + '">' +
                '    <i class="fa ' + statuses[event.message].icon + '"></i></div>' +
                '    <div class="timeline-message">' + escapeHtml(event.message) +
                '    <span class="timeline-date">' + moment.utc(event.time).local().format('MMMM Do YYYY h:mm:ss a') + '</span>'
            if (event.details) {
                details = JSON.parse(event.details)
                if (event.message == "Clicked Link" || event.message == "Submitted Data") {
                    deviceView = renderDevice(details)
                    if (deviceView) {
                        results += deviceView
                    }
                }
                if (event.message == "Submitted Data") {
                    results += '<div class="timeline-replay-button"><button onclick="replay(' + i + ')" class="btn btn-success">'
                    results += '<i class="fa fa-refresh"></i> Replay Credentials</button></div>'
                    results += '<div class="timeline-event-details"><i class="fa fa-caret-right"></i> View Details</div>'
                }
                if (details.payload) {
                    results += '<div class="timeline-event-results">'
                    results += '    <table class="table table-condensed table-bordered table-striped">'
                    results += '        <thead><tr><th>Parameter</th><th>Value(s)</tr></thead><tbody>'
                    $.each(Object.keys(details.payload), function (i, param) {
                        if (param == "rid") {
                            return true;
                        }
                        results += '    <tr>'
                        results += '        <td>' + escapeHtml(param) + '</td>'
                        results += '        <td>' + escapeHtml(details.payload[param]) + '</td>'
                        results += '    </tr>'
                    })
                    results += '       </tbody></table>'
                    results += '</div>'
                }
                if (details.error) {
                    results += '<div class="timeline-event-details"><i class="fa fa-caret-right"></i> View Details</div>'
                    results += '<div class="timeline-event-results">'
                    results += '<span class="label label-default">Error</span> ' + details.error
                    results += '</div>'
                }
            }
            results += '</div></div>'
        }
    })
    // Add the scheduled send event at the bottom
    if (record.status == "Scheduled" || record.status == "Retrying") {
        results += '<div class="timeline-entry">' +
            '    <div class="timeline-bar"></div>'
        results +=
            '    <div class="timeline-icon ' + statuses[record.status].label + '">' +
            '    <i class="fa ' + statuses[record.status].icon + '"></i></div>' +
            '    <div class="timeline-message">' + "Scheduled to send at " + record.send_date + '</span>'
    }
    results += '</div></div>'
    return results
}

var renderTimelineChart = function (chartopts) {
    return Highcharts.chart('timeline_chart', {
        chart: {
            zoomType: 'x',
            type: 'line',
            height: "200px"
        },
        title: {
            text: 'Campaign Timeline'
        },
        xAxis: {
            type: 'datetime',
            dateTimeLabelFormats: {
                second: '%l:%M:%S',
                minute: '%l:%M',
                hour: '%l:%M',
                day: '%b %d, %Y',
                week: '%b %d, %Y',
                month: '%b %Y'
            }
        },
        yAxis: {
            min: 0,
            max: 2,
            visible: false,
            tickInterval: 1,
            labels: {
                enabled: false
            },
            title: {
                text: ""
            }
        },
        tooltip: {
            formatter: function () {
                return Highcharts.dateFormat('%A, %b %d %l:%M:%S %P', new Date(this.x)) +
                    '<br>Event: ' + this.point.message + '<br>Email: <b>' + this.point.email + '</b>'
            }
        },
        legend: {
            enabled: false
        },
        plotOptions: {
            series: {
                marker: {
                    enabled: true,
                    symbol: 'circle',
                    radius: 3
                },
                cursor: 'pointer',
            },
            line: {
                states: {
                    hover: {
                        lineWidth: 1
                    }
                }
            }
        },
        credits: {
            enabled: false
        },
        series: [{
            data: chartopts['data'],
            dashStyle: "shortdash",
            color: "#cccccc",
            lineWidth: 1,
            turboThreshold: 0
        }]
    })
}

/* Renders a pie chart using the provided chartops */
var renderPieChart = function (chartopts) {
    return Highcharts.chart(chartopts['elemId'], {
        chart: {
            type: 'pie',
            events: {
                load: function () {
                    var chart = this,
                        rend = chart.renderer,
                        pie = chart.series[0],
                        left = chart.plotLeft + pie.center[0],
                        top = chart.plotTop + pie.center[1];
                    this.innerText = rend.text(chartopts['data'][0].count, left, top).
                    attr({
                        'text-anchor': 'middle',
                        'font-size': '24px',
                        'font-weight': 'bold',
                        'fill': chartopts['colors'][0],
                        'font-family': 'Helvetica,Arial,sans-serif'
                    }).add();
                },
                render: function () {
                    this.innerText.attr({
                        text: chartopts['data'][0].count
                    })
                }
            }
        },
        title: {
            text: chartopts['title']
        },
        plotOptions: {
            pie: {
                innerSize: '80%',
                dataLabels: {
                    enabled: false
                }
            }
        },
        credits: {
            enabled: false
        },
        tooltip: {
            formatter: function () {
                if (this.key == undefined) {
                    return false
                }
                return '<span style="color:' + this.color + '">\u25CF</span>' + this.point.name + ': <b>' + this.y + '%</b><br/>'
            }
        },
        series: [{
            data: chartopts['data'],
            colors: chartopts['colors'],
        }]
    })
}

/* Updates the bubbles on the map

@param {campaign.result[]} results - The campaign results to process
*/
var updateMap = function (results) {
    if (!map) {
        return
    }
    bubbles = []
    $.each(campaign.results, function (i, result) {
        // Check that it wasn't an internal IP
        if (result.latitude == 0 && result.longitude == 0) {
            return true;
        }
        newIP = true
        $.each(bubbles, function (i, bubble) {
            if (bubble.ip == result.ip) {
                bubbles[i].radius += 1
                newIP = false
                return false
            }
        })
        if (newIP) {
            bubbles.push({
                latitude: result.latitude,
                longitude: result.longitude,
                name: result.ip,
                fillKey: "point",
                radius: 2
            })
        }
    })
    map.bubbles(bubbles)
}

/**
 * Creates a status label for use in the results datatable
 * @param {string} status
 * @param {moment(datetime)} send_date
 */
function createStatusLabel(status, send_date) {
    var label = statuses[status].label || "label-default";
    var statusColumn = "<span class=\"label " + label + "\">" + status + "</span>"
    // Add the tooltip if the email is scheduled to be sent
    if (status == "Scheduled" || status == "Retrying") {
        var sendDateMessage = "Scheduled to send at " + send_date
        statusColumn = "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"top\" data-html=\"true\" title=\"" + sendDateMessage + "\">" + status + "</span>"
    }
    return statusColumn
}

function isScheduleEditable(result) {
    return result.status == "Scheduled" || result.status == "Retrying"
}

function buildScheduleMetadata(c) {
    scheduleMetadata = {
        scenarios: {},
        templates: {}
    }
    $("#schedule_filter_scenario").find("option:not(:first)").remove()
    $("#schedule_filter_template").find("option:not(:first)").remove()
    $.each(c.scenarios || [], function (i, scenario) {
        scheduleMetadata.scenarios[scenario.id] = scenario.name
        $("#schedule_filter_scenario").append($("<option>").val(scenario.id).text(scenario.name))
        $.each(scenario.templates || [], function (j, template) {
            scheduleMetadata.templates[template.id] = template.name
            $("#schedule_filter_template").append($("<option>").val(template.id).text(template.name))
        })
    })
}

function getScenarioName(result) {
    return scheduleMetadata.scenarios[result.scenario_id] || ("Scenario " + result.scenario_id)
}

function getTemplateName(result) {
    return scheduleMetadata.templates[result.template_id] || ("Template " + result.template_id)
}

function recipientName(result) {
    var name = $.trim((result.first_name || "") + " " + (result.last_name || ""))
    if (name.length > 0) {
        return name + " <" + result.email + ">"
    }
    return result.email
}

function toLocalDateTimeInputValue(date) {
    return moment(date).local().format("YYYY-MM-DDTHH:mm")
}

function scheduleRequestDate(value) {
    return moment(value).utc().format()
}

function selectedScheduleRows() {
    var rows = []
    $(".schedule-select:checked").each(function () {
        var rid = $(this).data("rid")
        var result = findResult(rid)
        if (result && isScheduleEditable(result)) {
            rows.push(result)
        }
    })
    return rows
}

function findResult(rid) {
    var found = null
    $.each(campaign.results || [], function (i, result) {
        if (result.id == rid) {
            found = result
            return false
        }
    })
    return found
}

function updateResultInCampaign(updated) {
    $.each(campaign.results || [], function (i, result) {
        if (result.id == updated.id) {
            campaign.results[i] = updated
            return false
        }
    })
}

function updateResultsTableRow(updated) {
    var table = $("#resultsTable").DataTable()
    table.rows().every(function (i) {
        var row = this.row(i)
        var rowData = row.data()
        if (rowData[0] == updated.id) {
            rowData[8] = moment(updated.send_date).format('MMMM Do YYYY, h:mm:ss a')
            rowData[7] = updated.reported
            rowData[6] = updated.status
            table.row(i).data(rowData)
            return false
        }
    })
    table.draw(false)
}

function scheduleActionButton(result) {
    if (!isScheduleEditable(result)) {
        return "<button class=\"btn btn-default btn-xs\" disabled><i class=\"fa fa-lock\"></i></button>"
    }
    return "<button class=\"btn btn-primary btn-xs\" onclick=\"openScheduleEdit('" + result.id + "')\" data-toggle=\"tooltip\" title=\"Edit send time\"><i class=\"fa fa-pencil\"></i></button>"
}

function renderScheduleSummary(results) {
    var scheduled = []
    var dayCounts = {}
    var now = moment()
    var startOfWeek = moment().startOf("week")
    var endOfWeek = moment().endOf("week")
    var todayCount = 0
    var weekCount = 0

    $.each(results, function (i, result) {
        if (!isScheduleEditable(result)) {
            return true
        }
        var sendDate = moment(result.send_date)
        scheduled.push(sendDate)
        if (sendDate.isSame(now, "day")) {
            todayCount++
        }
        if (sendDate.isBetween(startOfWeek, endOfWeek, null, "[]")) {
            weekCount++
        }
        var dayKey = sendDate.format("YYYY-MM-DD")
        dayCounts[dayKey] = (dayCounts[dayKey] || 0) + 1
    })

    scheduled.sort(function (a, b) {
        return a.valueOf() - b.valueOf()
    })

    var nextSend = "-"
    $.each(scheduled, function (i, sendDate) {
        if (sendDate.isSameOrAfter(now)) {
            nextSend = sendDate.format("MMM D, YYYY h:mm a")
            return false
        }
    })

    var busiestDay = "-"
    var busiestCount = 0
    $.each(dayCounts, function (day, count) {
        if (count > busiestCount) {
            busiestCount = count
            busiestDay = moment(day).format("MMM D, YYYY") + " (" + count + ")"
        }
    })

    $("#schedule_next_send").text(nextSend)
    $("#schedule_today").text(todayCount)
    $("#schedule_this_week").text(weekCount)
    $("#schedule_total").text(scheduled.length)
    $("#schedule_busiest_day").text(busiestDay)
}

function renderScheduleTable() {
    if (!scheduleTable) {
        return
    }
    var rows = []
    $.each(campaign.results || [], function (i, result) {
        var editable = isScheduleEditable(result)
        rows.push([
            editable ? "<input type=\"checkbox\" class=\"schedule-select\" data-rid=\"" + result.id + "\">" : "",
            moment(result.send_date).format("MMMM Do YYYY, h:mm:ss a"),
            escapeHtml(recipientName(result)),
            escapeHtml(getScenarioName(result)),
            escapeHtml(getTemplateName(result)),
            createStatusLabel(result.status, moment(result.send_date).format("MMMM Do YYYY, h:mm:ss a")),
            "<div class=\"text-right\">" + scheduleActionButton(result) + "</div>",
            result.id,
            result.send_date
        ])
    })
    scheduleTable.clear()
    scheduleTable.rows.add(rows)
    scheduleTable.draw()
    renderScheduleSummary(campaign.results || [])
    $('[data-toggle="tooltip"]').tooltip()
}

function applyScheduleFilters() {
    if (!scheduleTable) {
        return
    }
    var recipientFilter = $("#schedule_filter_recipient").val()
    var scenarioFilter = $("#schedule_filter_scenario").val()
    var templateFilter = $("#schedule_filter_template").val()
    scheduleTable.column(2).search(recipientFilter || "")
    scheduleTable.column(3).search(scenarioFilter ? "^" + $.fn.dataTable.util.escapeRegex(scheduleMetadata.scenarios[scenarioFilter]) + "$" : "", true, false)
    scheduleTable.column(4).search(templateFilter ? "^" + $.fn.dataTable.util.escapeRegex(scheduleMetadata.templates[templateFilter]) + "$" : "", true, false)
    scheduleTable.draw()
}

function openScheduleEdit(rid) {
    var result = findResult(rid)
    if (!result) {
        errorFlash("Schedule result not found")
        return
    }
    $("#schedule_edit_rid").val(result.id)
    $("#schedule_edit_send_date").val(toLocalDateTimeInputValue(result.send_date))
    $("#scheduleEditModal").modal("show")
}

function saveScheduleEdit() {
    var rid = $("#schedule_edit_rid").val()
    var sendDate = $("#schedule_edit_send_date").val()
    if (!sendDate) {
        errorFlash("Select a send date")
        return
    }
    saveScheduleChange(rid, scheduleRequestDate(sendDate))
        .success(function () {
            $("#scheduleEditModal").modal("hide")
        })
}

function saveScheduleChange(rid, sendDate) {
    return api.campaignId.scheduleResult(campaign.id, rid, { send_date: sendDate })
        .success(function (result) {
            updateResultInCampaign(result)
            updateResultsTableRow(result)
            renderScheduleTable()
            successFlashFade("Schedule updated", 3)
        })
        .error(function (data) {
            var message = "Error updating schedule"
            if (data.responseJSON && data.responseJSON.message) {
                message = data.responseJSON.message
            }
            errorFlash(message)
        })
}

function openBulkOffsetModal() {
    if (selectedScheduleRows().length == 0) {
        errorFlash("Select scheduled emails first")
        return
    }
    $("#scheduleBulkOffsetModal").modal("show")
}

function openBulkSetModal() {
    if (selectedScheduleRows().length == 0) {
        errorFlash("Select scheduled emails first")
        return
    }
    $("#schedule_bulk_set_send_date").val(toLocalDateTimeInputValue(selectedScheduleRows()[0].send_date))
    $("#scheduleBulkSetModal").modal("show")
}

function saveBulkOffset() {
    var minutes = parseInt($("#schedule_bulk_offset_minutes").val(), 10)
    if (isNaN(minutes)) {
        errorFlash("Enter a minute offset")
        return
    }
    var changes = $.map(selectedScheduleRows(), function (result) {
        return {
            rid: result.id,
            sendDate: moment(result.send_date).add(minutes, "minutes").utc().format()
        }
    })
    saveBulkScheduleChanges(changes, "#scheduleBulkOffsetModal")
}

function saveBulkSet() {
    var sendDate = $("#schedule_bulk_set_send_date").val()
    if (!sendDate) {
        errorFlash("Select a send date")
        return
    }
    var changes = $.map(selectedScheduleRows(), function (result) {
        return {
            rid: result.id,
            sendDate: scheduleRequestDate(sendDate)
        }
    })
    saveBulkScheduleChanges(changes, "#scheduleBulkSetModal")
}

function saveBulkScheduleChanges(changes, modalSelector) {
    var failures = []
    var requests = $.map(changes, function (change) {
        var deferred = $.Deferred()
        api.campaignId.scheduleResult(campaign.id, change.rid, { send_date: change.sendDate })
            .success(function (result) {
                updateResultInCampaign(result)
                updateResultsTableRow(result)
                deferred.resolve()
            })
            .error(function (data) {
                var message = "Error updating " + change.rid
                if (data.responseJSON && data.responseJSON.message) {
                    message = data.responseJSON.message
                }
                failures.push(message)
                deferred.resolve()
            })
        return deferred.promise()
    })

    $.when.apply($, requests).always(function () {
        $(modalSelector).modal("hide")
        renderScheduleTable()
        if (failures.length > 0) {
            errorFlash(failures.join("<br>"))
            return
        }
        successFlashFade("Schedule updated", 3)
    })
}

/* poll - Queries the API and updates the UI with the results
 *
 * Updates:
 * * Timeline Chart
 * * Email (Donut) Chart
 * * Map Bubbles
 * * Datatables
 */
function poll() {
    api.campaignId.results(campaign.id)
        .success(function (c) {
            campaign = c
            /* Update the timeline */
            var timeline_series_data = []
            $.each(campaign.timeline, function (i, event) {
                var event_date = moment.utc(event.time).local()
                timeline_series_data.push({
                    email: event.email,
                    message: event.message,
                    x: event_date.valueOf(),
                    y: 1,
                    marker: {
                        fillColor: statuses[event.message].color
                    }
                })
            })
            var timeline_chart = $("#timeline_chart").highcharts()
            timeline_chart.series[0].update({
                data: timeline_series_data
            })
            /* Update the results donut chart */
            var email_series_data = {}
            // Load the initial data
            Object.keys(statusMapping).forEach(function (k) {
                email_series_data[k] = 0
            });
            $.each(campaign.results, function (i, result) {
                email_series_data[result.status]++;
                if (result.reported) {
                    email_series_data['Email Reported']++
                }
                // Backfill status values
                var step = progressListing.indexOf(result.status)
                for (var i = 0; i < step; i++) {
                    email_series_data[progressListing[i]]++
                }
            })
            $.each(email_series_data, function (status, count) {
                var email_data = []
                if (!(status in statusMapping)) {
                    return true
                }
                email_data.push({
                    name: status,
                    y: Math.floor((count / campaign.results.length) * 100),
                    count: count
                })
                email_data.push({
                    name: '',
                    y: 100 - Math.floor((count / campaign.results.length) * 100)
                })
                var chart = $("#" + statusMapping[status] + "_chart").highcharts()
                chart.series[0].update({
                    data: email_data
                })
            })

            /* Update the datatable */
            resultsTable = $("#resultsTable").DataTable()
            resultsTable.rows().every(function (i, tableLoop, rowLoop) {
                var row = this.row(i)
                var rowData = row.data()
                var rid = rowData[0]
                $.each(campaign.results, function (j, result) {
                    if (result.id == rid) {
                        rowData[8] = moment(result.send_date).format('MMMM Do YYYY, h:mm:ss a')
                        rowData[7] = result.reported
                        rowData[6] = result.status
                        resultsTable.row(i).data(rowData)
                        if (row.child.isShown()) {
                            $(row.node()).find("#caret").removeClass("fa-caret-right")
                            $(row.node()).find("#caret").addClass("fa-caret-down")
                            row.child(renderTimeline(row.data()))
                        }
                        return false
                    }
                })
            })
            resultsTable.draw(false)
            renderScheduleTable()
            /* Update the map information */
            updateMap(campaign.results)
            $('[data-toggle="tooltip"]').tooltip()
            $("#refresh_message").hide()
            $("#refresh_btn").show()
        })
}

function checkUser() {
        api.user.current().success(function (u) {
            var permissions = {
                canDelete : false,
                canEdit : false,
            };
            permissions = CheckTeam(campaign.teams, u)

            var isOwner = false;
            if (u.id == campaign.user_id){
                var isOwner = true;
            }
            if (!(isOwner || permissions.canDelete)){
                $('#delete_button')[0].disabled = true;
            }
            if (!(isOwner || permissions.canEdit)){
                $('#complete_button')[0].disabled = true;
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error while getting user")
        })
}

function load() {
    campaign.id = window.location.pathname.split('/').slice(-1)[0]
    var use_map = JSON.parse(localStorage.getItem('gophish.use_map'))
    api.campaignId.get(campaign.id)
        .success(function (c) {
                campaign = c
                if (campaign) {
                    buildScheduleMetadata(campaign)
                    $("title").text(c.name + " - Gophish")
                    $("#loading").hide()
                    $("#campaignResults").show()
                    // Set the title
                    $("#page-title").text("Results for " + c.name)
                    if (c.status == "Completed") {
                        $('#complete_button')[0].disabled = true;
                        $('#complete_button').text('Completed!');
                        doPoll = false;
                    }
                    checkUser()
                    // Setup viewing the details of a result
                    $("#resultsTable").on("click", ".timeline-event-details", function () {
                        // Show the parameters
                        payloadResults = $(this).parent().find(".timeline-event-results")
                        if (payloadResults.is(":visible")) {
                            $(this).find("i").removeClass("fa-caret-down")
                            $(this).find("i").addClass("fa-caret-right")
                            payloadResults.hide()
                        } else {
                            $(this).find("i").removeClass("fa-caret-right")
                            $(this).find("i").addClass("fa-caret-down")
                            payloadResults.show()
                        }
                    })
                    // Setup the results table
                    resultsTable = $("#resultsTable").DataTable({
                        destroy: true,
                        "order": [
                            [2, "asc"]
                        ],
                        columnDefs: [{
                                orderable: false,
                                targets: "no-sort"
                            }, {
                                className: "details-control",
                                "targets": [1]
                            }, {
                                "visible": false,
                                "targets": [0, 8]
                            },
                            {
                                "render": function (data, type, row) {
                                    return createStatusLabel(data, row[8])
                                },
                                "targets": [6]
                            },
                            {
                                className: "text-center",
                                "render": function (reported, type, row) {
                                    if (type == "display") {
                                        if (reported) {
                                            return "<i class='fa fa-check-circle text-center text-success'></i>"
                                        }
                                        return "<i role='button' class='fa fa-times-circle text-center text-muted' onclick='report_mail(\"" + row[0] + "\", \"" + campaign.id + "\");'></i>"
                                    }
                                    return reported
                                },
                                "targets": [7]
                            }
                        ]
                    });
                    scheduleTable = $("#scheduleTable").DataTable({
                        destroy: true,
                        "order": [
                            [1, "asc"]
                        ],
                        columnDefs: [{
                                orderable: false,
                                targets: "no-sort"
                            }, {
                                "visible": false,
                                "targets": [7, 8]
                            }
                        ]
                    })
                    $("#schedule_filter_recipient").on("keyup change", applyScheduleFilters)
                    $("#schedule_filter_scenario").on("change", applyScheduleFilters)
                    $("#schedule_filter_template").on("change", applyScheduleFilters)
                    $("#schedule_select_all").on("change", function () {
                        $(".schedule-select").prop("checked", $(this).prop("checked"))
                    })
                    resultsTable.clear();
                    var email_series_data = {}
                    var timeline_series_data = []
                    Object.keys(statusMapping).forEach(function (k) {
                        email_series_data[k] = 0
                    });
                    $.each(campaign.results, function (i, result) {
                        resultsTable.row.add([
                            result.id,
                            "<i id=\"caret\" class=\"fa fa-caret-right\"></i>",
                            escapeHtml(result.first_name) || "",
                            escapeHtml(result.last_name) || "",
                            escapeHtml(result.email) || "",
                            escapeHtml(result.position) || "",
                            result.status,
                            result.reported,
                            moment(result.send_date).format('MMMM Do YYYY, h:mm:ss a')
                        ])
                        email_series_data[result.status]++;
                        if (result.reported) {
                            email_series_data['Email Reported']++
                        }
                        // Backfill status values
                        var step = progressListing.indexOf(result.status)
                        for (var i = 0; i < step; i++) {
                            email_series_data[progressListing[i]]++
                        }
                    })
                    resultsTable.draw();
                    renderScheduleTable()
                    // Setup tooltips
                    $('[data-toggle="tooltip"]').tooltip()
                    // Setup the individual timelines
                    $('#resultsTable tbody').on('click', 'td.details-control', function () {
                        var tr = $(this).closest('tr');
                        var row = resultsTable.row(tr);
                        if (row.child.isShown()) {
                            // This row is already open - close it
                            row.child.hide();
                            tr.removeClass('shown');
                            $(this).find("i").removeClass("fa-caret-down")
                            $(this).find("i").addClass("fa-caret-right")
                        } else {
                            // Open this row
                            $(this).find("i").removeClass("fa-caret-right")
                            $(this).find("i").addClass("fa-caret-down")
                            row.child(renderTimeline(row.data())).show();
                            tr.addClass('shown');
                        }
                    });
                    // Setup the graphs
                    $.each(campaign.timeline, function (i, event) {
                        if (event.message == "Campaign Created") {
                            return true
                        }
                        var event_date = moment.utc(event.time).local()
                        timeline_series_data.push({
                            email: event.email,
                            message: event.message,
                            x: event_date.valueOf(),
                            y: 1,
                            marker: {
                                fillColor: statuses[event.message].color
                            }
                        })
                    })
                    renderTimelineChart({
                        data: timeline_series_data
                    })
                    $.each(email_series_data, function (status, count) {
                        var email_data = []
                        if (!(status in statusMapping)) {
                            return true
                        }
                        email_data.push({
                            name: status,
                            y: Math.floor((count / campaign.results.length) * 100),
                            count: count
                        })
                        email_data.push({
                            name: '',
                            y: 100 - Math.floor((count / campaign.results.length) * 100)
                        })
                        var chart = renderPieChart({
                            elemId: statusMapping[status] + '_chart',
                            title: status,
                            name: status,
                            data: email_data,
                            colors: [statuses[status].color, '#dddddd']
                        })
                    })

                    if (use_map) {
                        $("#resultsMapContainer").show()
                        map = new Datamap({
                            element: document.getElementById("resultsMap"),
                            responsive: true,
                            fills: {
                                defaultFill: "#ffffff",
                                point: "#283F50"
                            },
                            geographyConfig: {
                                highlightFillColor: "#1abc9c",
                                borderColor: "#283F50"
                            },
                            bubblesConfig: {
                                borderColor: "#283F50"
                            }
                        });
                    }
                    updateMap(campaign.results)
                }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash(" Campaign not found!")
        })
}

var setRefresh

function refresh() {
    if (!doPoll) {
        return;
    }
    $("#refresh_message").show()
    $("#refresh_btn").hide()
    poll()
    clearTimeout(setRefresh)
    setRefresh = setTimeout(refresh, 60000)
};

function report_mail(rid, cid) {
    Swal.fire({
        title: "Are you sure?",
        text: "This result will be flagged as reported (RID: " + rid + ")",
        type: "question",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Continue",
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true
    }).then(function (result) {
        if (result.value){
            api.campaignId.get(cid).success((function(c) {
                report_url = new URL(c.url)
                report_url.pathname = '/report'
                report_url.search = "?rid=" + rid
                fetch(report_url)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP error! Status: ${response.status}`);
                    }
                    refresh();
                })
                .catch(error => {
                    let errorMessage = error.message;
                    if (error.message === "Failed to fetch") {
                        errorMessage = "This might be due to Mixed Content issues or network problems.";
                    }
                    Swal.fire({
                        title: 'Error',
                        text: errorMessage,
                        type: 'error',
                        confirmButtonText: 'Close'
                    });
                });
            }));
        }
    })
}

function dismiss() {
    $("#modal\\.flashes").empty();
    $("#name").val("");
}

$(document).ready(function () {
    load();

    // Start the polling loop
    setRefresh = setTimeout(refresh, 60000)

    // Setup multiple modals
    // Code based on http://miles-by-motorcycle.com/static/bootstrap-modal/index.html
    $('.modal').on('hidden.bs.modal', function (event) {
        $(this).removeClass('fv-modal-stack');
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') - 1);
    });
    $('.modal').on('shown.bs.modal', function (event) {
        // Keep track of the number of open modals
        if (typeof ($('body').data('fv_open_modals')) == 'undefined') {
            $('body').data('fv_open_modals', 0);
        }
        // if the z-index of this modal has been set, ignore.
        if ($(this).hasClass('fv-modal-stack')) {
            return;
        }
        $(this).addClass('fv-modal-stack');
        // Increment the number of open modals
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') + 1);
        // Setup the appropriate z-index
        $(this).css('z-index', 1040 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('.fv-modal-stack').css('z-index', 1039 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('fv-modal-stack').addClass('fv-modal-stack');
    });
    // above not needed

    $(document).on('hidden.bs.modal', '.modal', function () {
        $('.modal:visible').length && $(document.body).addClass('modal-open');
    });
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss();
    });

    // Select2 Defaults
    $.fn.select2.defaults.set("width", "100%");
    $.fn.select2.defaults.set("dropdownParent", $("#modal_body"));
    $.fn.select2.defaults.set("theme", "bootstrap");
    $.fn.select2.defaults.set("sorter", function (data) {
        return data.sort(function (a, b) {
            if (a.text.toLowerCase() > b.text.toLowerCase()) {
                return 1;
            }
            if (a.text.toLowerCase() < b.text.toLowerCase()) {
                return -1;
            }
            return 0;
        });
    })
})
