pipeline {
    agent any

    environment {
        APP_NAME = "backend-go"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Setup ENV') {
            steps {
                sh 'cp /home/dev/rebuild_asset/backend_go/.env .env'
            }
        }

        stage('Build Docker Image') {
            steps {
                sh 'make docker-build'
            }
        }

        stage('Deploy') {
            steps {
                sh 'make deploy'
            }
        }
    }

    post {
        success {
            echo '✅ Deploy berhasilss!'
        }
        failure {
            echo '❌ Deploy gagal!'
        }
    }
}