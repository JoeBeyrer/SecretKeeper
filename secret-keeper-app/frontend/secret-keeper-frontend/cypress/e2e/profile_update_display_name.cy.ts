describe('profile_update_display_name', () => {
  it('updates the display name and shows success', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/profile');

    cy.get('input[formControlName="display_name"]').clear().type('Alice Dobra');
    cy.get('.save-button').first().click();

    cy.get('.success-message').should('be.visible');

    cy.get('input[formControlName="display_name"]').clear().type('Alice');
    cy.get('.save-button').first().click();
  });
});
