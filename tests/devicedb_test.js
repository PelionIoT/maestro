const assert = require('assert');
const shell = require('shelljs');

const timeout = 10000;

let failure_count = 0;

afterEach(function(done) {
    this.timeout(timeout);
    if (!this.currentTest.state.includes('passed')) {
        done();
    } else {
        done();
    }
});

describe('DeviceDB', function() {

    before(function(done) {
        this.timeout(timeout);
        this.done = done;
        shell.exec("./scripts/maestro_startup.sh", {silent: true}, function(code, stdout, stderr) {
            this.done();
        }.bind(this));
    });

    after(function(done) {
        this.timeout(timeout);
        this.done = done;
        shell.exec("sudo pkill maestro", {silent: true}, function(code, stdout, stderr) {
            setTimeout(this.done, 2000);
        }.bind(this));
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

        it('should verify the default error filter works', function(done) {
            this.timeout(timeout);
            this.done = done;
            shell.exec("./scripts/test_devicedb.sh error", {silent: true}, function(code, stdout, stderr) {
                assert.equal(code, 0, stderr);
                this.done();
            }.bind(this));
        });

        it('should add the warn log filter to the file', function(done) {
            this.timeout(timeout);
            this.done = done;
            shell.exec("./scripts/test_devicedb.sh error warn", {silent: true}, function(code, stdout, stderr) {
                assert.equal(code, 0, stderr)
                this.done();
            }.bind(this));
        });

        it('should remove the error log filter to the file', function(done) {
            this.timeout(timeout);
            this.done = done;
            shell.exec("./scripts/test_devicedb.sh warn", {silent: true}, function(code, stdout, stderr) {
                assert.equal(code, 0, stderr)
                this.done();
            }.bind(this));
        });

        // it('should replace the warn log filter with the info log filter', function(done) {
        //     this.timeout(timeout);
        //     this.done = done;
        //     shell.exec("./scripts/test_devicedb.sh info", {silent: true}, function(code, stdout, stderr) {
        //         assert.equal(code, 0, stderr)
        //         this.done();
        //     }.bind(this));
        // });
    });
});

