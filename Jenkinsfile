def getReleaseVersion(String tagName) {
    if (tagName) {
        return tagName.replaceAll(/^v/, '')
    } else {
        return null
    }
}


pipeline {
  agent none

  environment {
    // Set RELEASE_VERSION only if TAG_NAME is set
    RELEASE_VERSION = getReleaseVersion(TAG_NAME)
  }

  stages {

        stage('Cleanup Workspace') {
          agent {
                node { label 'jenkinsworker00' }
            }
            options { skipDefaultCheckout() }
            steps {
                cleanWs()
            }
        }
        stage('Build') {
            agent {
                docker {
                  label 'jenkinsworker00'
                  image 'harbor.cloud.infn.it/jenkins-ci/go1.21.0:main'
                  reuseNode true
                }
            }
            steps {
                script {
                  sh '''
                  export GOCACHE=$WORKSPACE/.cache/go-build
                  export GOPATH=$WORKSPACE/go
                  make build
                  mv $GOPATH/bin/rclone $WORKSPACE/rclone_linux
                  '''
                }
            }
        }

        stage('Upload to Nexus'){
          when { tag "v*" }
          agent {
                node { label 'jenkinsworker00' }
            }
          options { skipDefaultCheckout() }
          steps{
            nexusArtifactUploader(
              nexusVersion: 'nexus3',
              protocol: 'https',
              nexusUrl: 'repo.cloud.cnaf.infn.it',
              version: RELEASE_VERSION,
              repository: 'rclone',
              groupId: '',
              credentialsId: 'nexus-credentials',
              artifacts: [
                  [ artifactId: 'rclone-linux', type: '', classifier: '', file: "rclone_linux" ],
              ]
            )

            }
            post {
               always {
                 cleanWs()
               }
             }
          }
  }
}