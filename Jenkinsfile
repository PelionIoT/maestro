pipeline {
  agent none
  options{
    skipDefaultCheckout()
  }
  stages {
    stage('Build') {
      agent{
        label 'noida-linux-ubuntu16-ci-slave'
      }
      steps {
        checkout scm
        withEnv(["GOROOT=/home/jenkins/go", "GOPATH=/home/jenkins/goprojects", "PATH+GO=/home/jenkins/goprojects/bin:/home/jenkins/go/bin"]){
          withCredentials([usernamePassword(credentialsId: 'ldap_credentials', passwordVariable: 'PASSWORD', usernameVariable: 'USERNAME')]) {
            sh "cd $HOME/goprojects/src/github.com/armPelionEdge && if [ ! -d \"maestro\" ]; then git clone https://github.com/sameer2209-arm/maestro.git && git checkout ${env.BRANCH_NAME} && go build; else cd maestro && git checkout ${env.BRANCH_NAME} && git pull origin ${env.BRANCH_NAME}; fi"
            sh "cd $HOME/goprojects/src/github.com/armPelionEdge && if [ ! -d \"greasego\" ]; then git clone https://github.com/armPelionEdge/greasego.git; else cd greasego && git pull; fi"
            sh 'cd /home/jenkins/goprojects/src/github.com/armPelionEdge/greasego && ./build-deps.sh && ./build.sh'
            //sh 'cd $HOME/goprojects/src/github.com/armPelionEdge/maestro && export GOBIN=$GOPATH/bin && export PATH=$GOBIN:$PATH && ./build-deps.sh && touch maestro.config && DEBUG=1 DEBUG2=1 ./build.sh && LD_LIBRARY_PATH=vendor/github.com/armPelionEdge/greasego/deps/lib maestro -config test-ok-string.config'
            sh 'cd /home/jenkins/goprojects/src/github.com/armPelionEdge/maestro && export GOBIN=$GOPATH/bin && export PATH=$GOBIN:$PATH && ./build-deps.sh && touch maestro.config && DEBUG=1 DEBUG2=1 ./build.sh'
          }
        }
      }
    }
    
    /*stage('Test and Code Review') {
      parallel {*/
        stage('Test'){
          agent{
            label 'noida-linux-ubuntu16-ci-slave'
          }
          steps {
            catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
              withEnv(["GOROOT=/home/jenkins/go", "GOPATH=/home/jenkins/goprojects", "PATH+GO=/home/jenkins/goprojects/bin:/home/jenkins/go/bin"]){
                sh 'go get -u github.com/jstemmer/go-junit-report'
                //sh 'cd $HOME/goprojects/src/github.com/armPelionEdge/fog-core && go test ./... -coverprofile cov.out -v 2>&1 | go-junit-report > report.xml && mv report.xml ~/workspace/\"Fog-Core CI Pipeline\"/'
                //sh 'cd networking && sudo -E env PATH=/var/lib/jenkins/go/bin:/var/lib/jenkins/goprojects/bin:$PATH  LD_LIBRARY_PATH=../vendor/github.com/armPelionEdge/greasego/deps/lib go test -v -run DhcpRequest -coverprofile cov.out -v 2>&1 | go-junit-report > report.xml'
                sh "cd /home/jenkins/goprojects/src/github.com/armPelionEdge/maestro && env LD_LIBRARY_PATH=../vendor/github.com/armPelionEdge/greasego/deps/lib go test -v -coverprofile cov.out 2>&1 ./... | go-junit-report > report.xml && mv report.xml /home/jenkins/workspace/maestro_${env.BRANCH_NAME}"
                sh "cd /home/jenkins/goprojects/src/github.com/armPelionEdge/maestro && gocover-cobertura < cov.out > coverage.xml && mv coverage.xml /home/jenkins/workspace/maestro_${env.BRANCH_NAME}"
              }
            }
          }
        }
        
        /*stage('SonarQube'){
          agent{
            label 'master'
          }
          environment {
            scannerHome = tool 'SonarQubeScanner'
          }
          steps {
            withSonarQubeEnv('sonarqube') {
              sh "cd $HOME/goprojects/src/github.com/armPelionEdge/maestro && ${scannerHome}/bin/sonar-scanner && rsync -a .scannerwork /var/lib/jenkins/workspace/maestro_master@2/"
            }
          }
        }
      }
    }*/
    
    /*stage('Utils'){
      steps{
        sh 'sshpass -p "jenkins" scp jenkins@10.166.150.42:/home/jenkins/goprojects/src/github.com/armPelionEdge/maestro/report.xml /var/lib/jenkins/workspace/maestro_master/'
        sh 'sshpass -p "jenkins" scp jenkins@10.166.150.42:/home/jenkins/goprojects/src/github.com/armPelionEdge/maestro/coverage.xml /var/lib/jenkins/workspace/maestro_master/'
        sh 'sshpass -p "jenkins" scp jenkins@10.166.150.42:/home/jenkins/goprojects/src/github.com/armPelionEdge/maestro/cov.out /var/lib/jenkins/goprojects/src/github.com/armPelionEdge/maestro/'
      }
    }*/
    
   stage('Auto Doc') {
     agent{
        label 'noida-linux-ubuntu16-ci-slave'
      }
      steps {
        withEnv(["GOROOT=$HOME/go", "GOPATH=$HOME/goprojects", "PATH+GO=$HOME/goprojects/bin:$HOME/go/bin"]){
          sh 'go get github.com/robertkrimen/godocdown/godocdown'
          sh 'cd /home/jenkins/goprojects/src/github.com/armPelionEdge/maestro && go list ./... > maestro_packages.txt'
          sh 'cd /home/jenkins/goprojects/src/github.com/armPelionEdge/maestro && input=maestro_packages.txt && while IFS= read -r line; do godocdown -plain=false $line >> maestro_docs.md; done < $input && mv maestro_docs.md /home/jenkins/workspace/maestro_master/maestro_docs.md'
        }
      }
    }
    
    /*stage('SonarQube'){
      environment {
        scannerHome = tool 'SonarQubeScanner'
      }
      steps {
        withSonarQubeEnv('sonarqube') {
            sh "cd $HOME/goprojects/src/github.com/armPelionEdge/fog-core && ${scannerHome}/bin/sonar-scanner"
        }
      }
    }*/
  }
  
  post{
    success{
      slackSend(channel: '#edge-jenkins-ci', color: 'good', message: "JOB NAME: ${env.JOB_NAME}\nBUILD NUMBER: ${env.BUILD_NUMBER}\nSTATUS: ${currentBuild.currentResult}\n${env.RUN_DISPLAY_URL}")
    }
    failure{
      slackSend(channel: '#edge-jenkins-ci', color: 'danger', message: "JOB NAME: ${env.JOB_NAME}\nBUILD NUMBER: ${env.BUILD_NUMBER}\nSTATUS: ${currentBuild.currentResult}\n${env.RUN_DISPLAY_URL}")
    }
    unstable{
      slackSend(channel: '#edge-jenkins-ci', color: 'warning', message: "JOB NAME: ${env.JOB_NAME}\nBUILD NUMBER: ${env.BUILD_NUMBER}\nSTATUS: ${currentBuild.currentResult}\n${env.RUN_DISPLAY_URL}")
    }
    always{
      node('noida-linux-ubuntu16-ci-slave'){
      junit 'report.xml'
      step([$class: 'CoberturaPublisher', autoUpdateHealth: false, autoUpdateStability: false, coberturaReportFile: 'coverage.xml', failUnhealthy: false, failUnstable: false, maxNumberOfBuilds: 0, onlyStable: false, sourceEncoding: 'ASCII', zoomCoverageChart: false])
      archiveArtifacts artifacts: 'maestro_docs.md'
      //sh "curl -X POST -H \"Content-type: application/json\" --data '{\"channel\": \"#edge-jenkins-ci\", \"username\": \"webhookbot\", \"text\": \"JOB NAME: ${env.JOB_NAME}\\nBUILD NUMBER: ${env.BUILD_NUMBER}\\nSTATUS: ${currentBuild.currentResult}\\n${env.RUN_DISPLAY_URL}\"}' https://hooks.slack.com/services/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"
    }
   }
 }
}
