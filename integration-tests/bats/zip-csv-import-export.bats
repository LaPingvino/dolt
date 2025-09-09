#!/usr/bin/env bats
load $BATS_TEST_DIRNAME/helper/common.bash

setup() {
    setup_common

    # Create test CSV data
    cat <<EOF > sample.csv
id,name,age,city
1,John Doe,25,New York
2,Jane Smith,30,Los Angeles
3,Bob Johnson,35,Chicago
EOF

    # Create GTFS test files
    cat <<EOF > agency.txt
agency_id,agency_name,agency_url,agency_timezone
MTA,Metropolitan Transportation Authority,https://www.mta.info,America/New_York
EOF

    cat <<EOF > stops.txt
stop_id,stop_name,stop_lat,stop_lon
1001,Main St & 1st Ave,40.7589,-73.9851
1002,Central Station,40.7614,-73.9776
EOF

    cat <<EOF > routes.txt
route_id,agency_id,route_short_name,route_long_name,route_type
R1,MTA,1,Broadway Line,1
EOF

    cat <<EOF > trips.txt
route_id,service_id,trip_id,trip_headsign,direction_id
R1,WEEKDAY,T001,Uptown,0
EOF

    cat <<EOF > stop_times.txt
trip_id,arrival_time,departure_time,stop_id,stop_sequence
T001,06:00:00,06:00:00,1001,1
T001,06:05:00,06:05:00,1002,2
EOF

    # Create ZIP files for testing
    if command -v zip >/dev/null 2>&1; then
        zip sample_csv.zip sample.csv >/dev/null
        zip gtfs_sample.zip agency.txt stops.txt routes.txt trips.txt stop_times.txt >/dev/null
    else
        skip "zip command not available"
    fi
}

teardown() {
    teardown_common
    rm -f *.csv *.txt *.zip
}

@test "zip-csv-import-export: import CSV from ZIP file" {
    dolt table import -c users sample_csv.zip
    run dolt sql -q "SELECT COUNT(*) as count FROM users"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "3" ]]

    run dolt sql -q "SELECT name FROM users WHERE id = 1"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "John Doe" ]]
}

@test "zip-csv-import-export: export table to ZIP file" {
    dolt sql -q "CREATE TABLE test_table (id INT PRIMARY KEY, name VARCHAR(50))"
    dolt sql -q "INSERT INTO test_table VALUES (1, 'Alice'), (2, 'Bob')"

    dolt table export test_table exported.zip
    [ -f exported.zip ]

    if command -v unzip >/dev/null 2>&1; then
        run unzip -l exported.zip
        [ "$status" -eq 0 ]
        [[ "$output" =~ "test_table.csv" ]]
    fi
}

@test "zip-csv-import-export: GTFS file detection and import" {
    dolt table import -c transit gtfs_sample.zip
    run dolt sql -q "SELECT COUNT(*) as count FROM transit"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "7" ]] # Total rows from all GTFS files

    # Verify some GTFS data was imported
    run dolt sql -q "SELECT * FROM transit LIMIT 1"
    [ "$status" -eq 0 ]
}

@test "zip-csv-import-export: import with CSV options" {
    # Test with delimiter and no-header options
    cat <<EOF > delimited.csv
1|John|25
2|Jane|30
EOF

    if command -v zip >/dev/null 2>&1; then
        zip delimited.zip delimited.csv >/dev/null

        dolt table import -c users_delimited delimited.zip --delim="|" --no-header --columns="id,name,age"
        run dolt sql -q "SELECT name FROM users_delimited WHERE id = 1"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "John" ]]
    else
        skip "zip command not available"
    fi
}

@test "zip-csv-import-export: round-trip import and export" {
    # Import original data
    dolt table import -c original sample_csv.zip

    # Export to new ZIP
    dolt table export original roundtrip.zip

    # Import exported data to new table
    dolt table import -c roundtrip roundtrip.zip

    # Verify data matches
    run dolt sql -q "SELECT COUNT(*) FROM original o JOIN roundtrip r ON o.id = r.id WHERE o.name = r.name AND o.age = r.age AND o.city = r.city"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "3" ]]
}

@test "zip-csv-import-export: file type parameter" {
    # Test explicit file type specification
    cp sample_csv.zip sample.unknown

    dolt table import -c explicit sample.unknown --file-type=zip
    run dolt sql -q "SELECT COUNT(*) FROM explicit"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "3" ]]
}
