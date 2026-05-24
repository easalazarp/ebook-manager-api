# E-book Management API - Functional Go & Supabase

## 📌 Descripción del Proyecto
Este proyecto consiste en una **API REST robusta** diseñada para la gestión integral de libros electrónicos (E-books). El sistema permite administrar catálogos bibliográficos, gestionar autores y categorías, y simular el flujo de descargas de archivos digitales [1].

Desarrollado bajo un enfoque de **Ingeniería de Software Senior**, este sistema se presenta como un **Producto Mínimo Viable (MVP) centrado exclusivamente en el Backend** [1, 2]. Esta decisión estratégica permite que un único desarrollador asuma con éxito una carga de trabajo originalmente planificada para un equipo de cuatro integrantes, priorizando la lógica de negocio y la calidad del código sobre la interfaz de usuario [1, 2].

## 🛠️ Stack Tecnológico
*   **Lenguaje:** [Go (Golang)](https://go.dev/) - Seleccionado por su eficiencia y soporte para concurrencia [2].
*   **BaaS (Backend as a Service):** [Supabase](https://supabase.com/) - Provee la base de datos PostgreSQL, autenticación JWT y almacenamiento de archivos (Storage) [3, 4].
*   **Librerías Core:** 
    *   `net/http` para el servidor web.
    *   `google/uuid` para identificadores inmutables.
    *   SDK oficial de Supabase para Go [3].

## 🏗️ Arquitectura del Sistema
El proyecto sigue una **Arquitectura Modular Limpia**, separando las responsabilidades para facilitar el mantenimiento y la escalabilidad [4]:

*   `cmd/api/`: Punto de entrada de la aplicación.
*   `internal/handlers/`: Controladores de rutas HTTP.
*   `internal/service/`: Lógica de negocio pura (Paradigma Funcional).
*   `internal/repository/`: Interacción con Supabase/PostgreSQL.
*   `internal/models/`: Estructuras de datos inmutables.

## 🚀 Instalación y Configuración
Para inicializar el entorno de desarrollo y las dependencias, ejecute:

```bash
# Inicializar el módulo de Go
go mod init ebook-manager-api

# Descargar dependencias
go mod tidy
🧩 Módulos Implementados
Gestión de Catálogo: CRUD completo de libros, autores y categorías.
Descargas e Historial: Lógica para obtención de archivos PDF/EPUB y registro de logs de actividad.
🛡️ Enfoque Funcional
El sistema implementa Programación Funcional mediante el uso de funciones puras, inmutabilidad de estados y funciones de orden superior para middlewares de seguridad, garantizando un código predecible y libre de efectos secundarios.
