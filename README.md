stailf
======
a stupid simple(and fairly untested!) tool to tail your Splunk logs from the comfort of your own shell
Might not do the right thing&trade; but heck it works(?)

## Usage

    # Get a session key from Splunk
    curl -k -d username="admin" -d password="changeme" https://localhost:8089/services/auth/login
    ...
    <response>
        <sessionKey>I5i_6dCMN0JxxvUOp_bovG8Obye_6zd0HJS^Pwyti_dGWpv0L0tsdDW9VWR56cs0h055lMI2hTmdwVvQNkIePVDyjvQ1sAardhKPXu6smF</sessionKey>
    </response>

    # Set environment variables and run
    export SPLUNK_URL=https://localhost:8089
    export SPLUNK_SESSION="I5i_6dCMN0JxxvUOp_bovG8Obye_6zd0HJS^Pwyti_dGWpv0L0tsdDW9VWR56cs0h055lMI2hTmdwVvQNkIePVDyjvQ1sAardhKPXu6smF"
    
    stailf "source=\"/var/log/syslog\" *"


