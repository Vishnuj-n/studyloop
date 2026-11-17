# Release v1.25.0 - Maintenance & Refactoring

Internal code maintenance and cleanup:
- **Database & State**: Removed obsolete `skip_to_reading_active` setting/column and cleaned up dead dashboard UI references.
- **Dependencies**: Pruned unused `@clerk/vue` package and cleaned lockfile.
- **API & Dead Code**: Removed redundant API aliases in `appApi.js` and dead exports across frontend services.
- **Component Refactoring**: Restructured `Flashcards.vue` and `Socratic.vue`, extracted `SettingsProfileModal.vue`, and added unit test coverage.
