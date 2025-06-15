#version 460 core
in vec2 TexCoord;
in vec3 FragPos;
in vec3 Normal;
out vec4 FragColor;

uniform sampler2D ourTexture;
uniform vec3 lightPos;
uniform vec3 lightColor;
uniform float ambient;
uniform float specularStrength;
uniform float shininess;
uniform vec3 viewPos;

void main()
{
    // Éclairage local
    vec3 ambientTerm = ambient * lightColor;
    vec3 norm = normalize(Normal);
    vec3 lightDir = normalize(lightPos - FragPos);
    float diff = max(dot(norm, lightDir), 0.0);
    vec3 diffuse = diff * lightColor;
    vec3 viewDir = normalize(viewPos - FragPos);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), shininess);
    vec3 specular = specularStrength * spec * lightColor;
    
    // Atténuation basée sur la distance
    float distance = length(lightPos - FragPos);
    float attenuation = 1.0 - smoothstep(5.0, 15.0, distance);
    
    // Combinaison des composantes d'éclairage
    vec3 finalLight = ambientTerm + attenuation * (diffuse + specular);
    
    // Luminosité minimale
    finalLight = max(finalLight, vec3(0.5));
    
    // Application de la texture
    vec4 texColor = texture(ourTexture, TexCoord);
    FragColor = vec4(finalLight, 1.0) * texColor;
}
