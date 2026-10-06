FROM eclipse-temurin:21-jre

WORKDIR /app

COPY . .

CMD ["java", "-Djava.library.path=./DynamoDBLocal_lib", "-jar", "DynamoDBLocal.jar", "-sharedDb"]